# Segurança no Backend — Guia Teórico

Este documento explica os problemas de segurança corrigidos no backend, a teoria por trás de cada um, e como foram resolvidos.

---

## 1. Race Condition (Condição de Corrida)

### O que é

Uma race condition acontece quando **dois ou mais processos acessam e modificam um recurso compartilhado ao mesmo tempo**, e o resultado depende da ordem de execução.

### Como funcionava antes (vulnerável)

```
Requisição A                    Requisição B
────────────                    ────────────
Lê saldo: R$110                 Lê saldo: R$110    ← ambos lêem o mesmo valor
Verifica: 110 >= 103? ✅        Verifica: 110 >= 103? ✅
Subtrai: 110 - 103 = 7          Subtrai: 110 - 103 = 7
Salva saldo: R$7                Salva saldo: R$7    ← deveria dar erro!
```

Resultado: o usuário usou o serviço **2 vezes** pagando **1 vez**. Perdemos R$103,56.

### Por que acontece

O problema é que a operação "ler → verificar → subtrair → salvar" não é **atômica**. Entre a leitura e a escrita, outro processo pode interferir.

### Como foi resolvido

**Operação atômica** — toda a lógica de "ler saldo, verificar, subtrair" acontece dentro de uma única operação indivisível.

#### Em memória (desenvolvimento)

Usamos um **mutex** (mutual exclusion). O mutex é como uma chave: só quem tem a chave pode entrar na sala. Os outros esperam na fila.

```go
func (r *UserRepository) DebitBalance(ctx context.Context, id string, amount int64) error {
    r.mu.Lock()         // Tranca — ninguém mais entra
    defer r.mu.Unlock() // Destranca ao sair

    entity := r.items[id]
    if entity.Balance < amount {
        return ErrInsufficientBalance  // Sem saldo, sai sem alterar nada
    }
    entity.Balance -= amount           // Subtrai
    r.items[id] = entity               // Salva
    return nil
}
```

Agora:
```
Requisição A                    Requisição B
────────────                    ────────────
Lock ✅                          Lock ⏳ (esperando)
Lê saldo: R$110
Subtrai: 110 - 103 = 7
Salva: R$7
Unlock
                                 Lock ✅
                                 Lê saldo: R$7
                                 Verifica: 7 >= 103? ❌
                                 Retorna erro
                                 Unlock
```

#### No Firestore (produção)

Usamos **transações** do Firestore. Uma transação garante que se dois processos tentarem modificar o mesmo documento ao mesmo tempo, um deles será automaticamente rejeitado e precisa tentar novamente.

```go
func (r *UserRepository) DebitBalance(ctx context.Context, id string, amount int64) error {
    ref := r.client.Collection("users").Doc(id)
    return r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
        snap, _ := tx.Get(ref)        // Lê dentro da transação
        entity := parseUser(snap)
        if entity.Balance < amount {
            return ErrInsufficientBalance
        }
        entity.Balance -= amount
        return tx.Set(ref, entity)     // Salva dentro da transação
    })
}
```

O Firestore usa **optimistic locking**: ele permite que a transação execute, mas na hora de salvar, verifica se o documento foi alterado por outra transação. Se foi, ele **rejeita** e tenta novamente automaticamente.

### Conceitos relacionados

- **Atomicidade** — uma operação é atômica quando é "tudo ou nada". Ou todas as etapas acontecem, ou nenhuma acontece.
- **Mutex** — mecanismo de exclusão mútua. Garante que apenas uma goroutine por vez execute uma seção crítica.
- **Transação** — no banco de dados, é um grupo de operações que são executadas como uma unidade atômica. Se qualquer parte falhar, todas são revertidas.
- **Optimistic Locking** — permite acesso concorrente mas detecta conflitos na hora de salvar. Usado pelo Firestore.
- **Pessimistic Locking** — bloqueia o recurso antes de modificar (como o mutex). Usado na implementação em memória.

---

## 2. Rollback (Reversão de Débito)

### O que é

Rollback é a capacidade de **desfazer uma operação** quando algo dá errado depois que ela já foi executada.

### Como funcionava antes (vulnerável)

```
1. Debita R$103,56 do saldo do usuário  ✅
2. Chama API da Infovist               ❌ API fora do ar!
3. Usuário perdeu dinheiro sem receber o serviço
```

### Como foi resolvido

```
1. Debita R$103,56 do saldo do usuário  ✅
2. Chama API da Infovist               ❌ API fora do ar!
3. Credita R$103,56 de volta (rollback) ✅
4. Retorna erro ao usuário
```

```go
func (s *InfovistService) CreateInspection(ctx context.Context, userID string, input ...) {
    // 1. Debita
    s.userRepo.DebitBalance(ctx, userID, s.costCreateInspection)

    // 2. Tenta chamar API
    result, err := s.client.CreateInspection(ctx, token, input)
    if err != nil {
        // 3. Falhou — rollback
        s.userRepo.CreditBalance(ctx, userID, s.costCreateInspection)
        return nil, err
    }

    return result, nil
}
```

### Por que não debitar só depois do sucesso?

Porque entre o "verificar saldo" e o "debitar", outro request poderia passar (race condition novamente). A abordagem correta é:

1. **Debitar atomicamente** (garante reserva do saldo)
2. **Executar a operação** (chamar API externa)
3. **Se falhar, creditar de volta** (rollback atômico)

### Conceitos relacionados

- **Compensating Transaction** — quando não é possível fazer rollback automático (como em bancos de dados), você executa uma "transação compensatória" que desfaz o efeito. No nosso caso, o CreditBalance é a compensação do DebitBalance.
- **Saga Pattern** — em sistemas distribuídos, cada serviço executa sua operação local e, em caso de falha, executa a compensação. É o que fazemos: debitamos localmente, chamamos o serviço externo, e compensamos se falhar.

---

## 3. Endpoint de Crédito Direto Exposto

### O que é

Um endpoint que permite adicionar saldo sem pagamento real. Útil para testes, perigoso em produção.

### O risco

```
POST /payments/users/meu-id/credit
{ "amount": 99999999 }
```

Se esse endpoint funcionar em produção, qualquer usuário autenticado pode dar a si mesmo saldo infinito.

### Como foi resolvido

O handler agora recebe uma flag `allowCredit` que é `true` apenas quando o sistema está em modo mock (sem PicPay configurado).

```go
type PaymentHandler struct {
    service     PaymentService
    allowCredit bool           // true apenas em mock/dev
}

func (h *PaymentHandler) Credit(c *gin.Context) {
    if !h.allowCredit {
        c.JSON(403, gin.H{"error": "direct credit is disabled in production"})
        return
    }
    // ... lógica normal
}
```

No wiring:

```go
isMockPayment := cfg.PicPayClientID == "" || cfg.PicPayClientSecret == ""
paymentHandler := NewPaymentHandler(paymentService, isMockPayment)
```

### Conceitos relacionados

- **Principle of Least Privilege** — cada componente deve ter apenas as permissões mínimas necessárias. Em produção, o endpoint de crédito direto não é necessário, então é desabilitado.
- **Feature Flag** — uma flag que controla se uma funcionalidade está ativa. Neste caso, `allowCredit` controla se o crédito direto está disponível.
- **Defense in Depth** — múltiplas camadas de proteção. Mesmo que o endpoint exista, ele verifica a flag antes de executar.

---

## 4. IDOR — Insecure Direct Object Reference

### O que é

IDOR acontece quando um sistema permite que um usuário **acesse recursos de outro usuário** simplesmente mudando um identificador na requisição (como um ID ou reference_id).

### O risco (SyncOrder)

```
POST /payments/orders/REFERENCIA-DO-OUTRO/sync
```

O endpoint de sync forçava a verificação de pagamento no PicPay para qualquer pedido, sem verificar se pertencia ao usuário autenticado. Um atacante poderia:

1. Criar seu próprio pedido e observar o padrão do `reference_id`
2. Tentar `reference_id`s de outros usuários
3. Forçar o processamento de pagamento de pedidos alheios

### Como foi resolvido

Adicionado `ProcessWebhookForUser` que verifica ownership:

```go
func (s *PaymentService) ProcessWebhookForUser(ctx context.Context, referenceID, userID string) error {
    order, _ := s.orderRepo.GetByReferenceID(ctx, referenceID)

    // Verifica se o pedido pertence ao usuário
    if order.UserID != userID {
        return errors.New("order does not belong to this user")
    }

    return s.ProcessWebhook(ctx, referenceID)
}
```

O webhook público (`POST /payments/webhook`) continua sem essa verificação porque é chamado pelo PicPay (servidor-a-servidor), não pelo usuário.

### Conceitos relacionados

- **IDOR (Insecure Direct Object Reference)** — está no OWASP Top 10. Acontece quando a aplicação expõe referências internas (IDs, nomes de arquivo) e não verifica se o usuário tem permissão para acessá-las.
- **Authorization vs Authentication** — autenticação verifica "quem é você", autorização verifica "o que você pode fazer". O SyncOrder tinha autenticação (JWT), mas faltava autorização (verificar se o recurso pertence ao usuário).
- **Ownership Check** — verificar se o recurso solicitado pertence ao usuário que fez a requisição. Deve ser feito em TODOS os endpoints que acessam recursos por ID.

---

## 5. Idempotência

### O que é

Uma operação é **idempotente** quando executá-la 1 vez ou N vezes produz o mesmo resultado. Isso é essencial para webhooks e pagamentos.

### O risco

Se o PicPay chamar o webhook 2 vezes para o mesmo pagamento (o que é normal — webhooks não garantem entrega única), o saldo seria creditado 2 vezes.

### Como funciona no código

```go
func (s *PaymentService) ProcessWebhook(ctx context.Context, referenceID string) error {
    order, _ := s.orderRepo.GetByReferenceID(ctx, referenceID)

    // Já processado — não faz nada
    if order.Status == payment.StatusPaid {
        return nil
    }

    // Re-verifica com o PicPay (nunca confia no payload do webhook)
    status, _ := s.provider.GetOrderStatus(ctx, referenceID)

    order.Status = status
    s.orderRepo.Update(ctx, order)

    if status == payment.StatusPaid {
        s.userRepo.CreditBalance(ctx, order.UserID, order.AmountCents)
    }

    return nil
}
```

Três proteções:
1. **Check de status** — se já está pago, retorna sem fazer nada
2. **Re-verificação** — nunca confia no payload do webhook, sempre consulta o PicPay
3. **CreditBalance atômico** — mesmo que dois webhooks passem da verificação, o crédito é atômico

### Conceitos relacionados

- **Idempotência** — f(x) = f(f(x)). Aplicar a operação múltiplas vezes tem o mesmo efeito que aplicar uma vez.
- **At-least-once delivery** — webhooks podem ser entregues mais de uma vez. O receptor deve ser idempotente.
- **Never trust the client** — nunca confie nos dados enviados pelo webhook. Sempre re-verifique com a fonte (PicPay).

---

## 6. Validação antes do Débito

### O que é

Validar os dados de entrada **antes** de qualquer operação com efeito colateral (como debitar saldo).

### Por que importa

Se a validação falhar depois do débito, o usuário perde dinheiro por um erro de preenchimento.

### Como está no código

```go
func (s *InfovistService) CreateInspection(ctx context.Context, userID string, input ...) {
    // 1. Validação PRIMEIRO
    if input.Customer == "" {
        return nil, errors.New("customer is required")  // Sem débito
    }
    if input.Plate == "" && input.Chassis == "" {
        return nil, errors.New("plate or chassis is required")  // Sem débito
    }

    // 2. Débito SÓ DEPOIS da validação
    s.userRepo.DebitBalance(ctx, userID, s.costCreateInspection)

    // 3. Chamada à API
    // ...
}
```

### Conceitos relacionados

- **Fail Fast** — detectar erros o mais cedo possível, antes de executar operações caras ou irreversíveis.
- **Input Validation** — sempre validar dados na entrada do sistema. Nunca confiar que o cliente enviou dados corretos.

---

## Testes de Segurança

Todos esses cenários são cobertos por testes unitários automatizados:

| Teste | O que garante |
|---|---|
| `TestDebitBalance_ConcurrentRace` | 50 goroutines tentam debitar — só 1 consegue |
| `TestDebitBalance_ConcurrentMultiple` | 200 goroutines debitam R$1 de R$100 — exatamente 100 passam |
| `TestCreditBalance_AtomicConcurrent` | 100 créditos simultâneos somam corretamente |
| `TestCreateInspection_RollbackOnAPIFailure` | API falha → saldo restaurado |
| `TestGetReportV2_RollbackOnAPIFailure` | API falha → saldo restaurado |
| `TestProcessWebhookForUser_WrongUser` | Outro usuário não consegue sync |
| `TestProcessWebhook_IdempotentCredit` | Webhook duplicado não credita 2x |
| `TestCreateInspection_ValidationErrors` | Validação roda antes do débito |
| `TestCreateInspection_InsufficientBalance` | Sem saldo = sem débito |

Para rodar:

```bash
go test ./internal/infra/memory/ ./internal/app/ -v
```

---

## Referências para estudo

- **OWASP Top 10** — lista das 10 vulnerabilidades mais comuns em aplicações web
- **Race Conditions in Go** — https://go.dev/doc/articles/race_detector
- **Firestore Transactions** — operações atômicas no Firestore
- **Saga Pattern** — padrão para transações distribuídas com compensação
- **IDOR** — Insecure Direct Object Reference (OWASP A01:2021)
