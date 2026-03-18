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

## Melhorias futuras

### 7. Rate Limiting por Usuário

#### O que é

Rate limiting é uma técnica que **limita a quantidade de requisições** que um usuário pode fazer em um período de tempo. Sem isso, um usuário (ou atacante) pode:

- **Spam de consultas** — disparar centenas de requests por segundo, consumindo saldo rapidamente (pode ser intencional ou um bug no front)
- **Ataque de força bruta** — tentar adivinhar protocolos de vistorias tentando milhares de combinações
- **DDoS** — sobrecarregar a API com volume de requisições, afetando todos os usuários
- **Abuso financeiro** — se houver alguma falha no débito, o volume alto de requests aumenta a chance de explorar

#### Como implementar

A abordagem mais comum é o **Token Bucket** (balde de fichas):

```
Cada usuário tem um balde com N fichas.
Cada request consome 1 ficha.
Fichas são repostas a uma taxa fixa (ex: 10 por minuto).
Se o balde está vazio, request é rejeitado com 429 Too Many Requests.
```

##### Opção 1 — Middleware no Gin (em memória)

```go
// Exemplo conceitual — implementar em internal/interfaces/http/rate_limiter.go
type RateLimiter struct {
    mu       sync.Mutex
    buckets  map[string]*bucket  // userID → bucket
    rate     int                 // fichas por minuto
    capacity int                 // máximo de fichas acumuladas
}

type bucket struct {
    tokens    float64
    lastCheck time.Time
}

func (rl *RateLimiter) Allow(userID string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    b := rl.buckets[userID]
    now := time.Now()
    elapsed := now.Sub(b.lastCheck).Minutes()

    // Repor fichas proporcionalmente ao tempo passado
    b.tokens = min(float64(rl.capacity), b.tokens + elapsed*float64(rl.rate))
    b.lastCheck = now

    if b.tokens < 1 {
        return false  // Sem fichas — rejeitar
    }
    b.tokens--
    return true
}
```

Uso como middleware:
```go
func (rl *RateLimiter) Handler() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, _ := GetAuthUserID(c)
        if !rl.Allow(userID) {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "too many requests, try again later",
            })
            return
        }
        c.Next()
    }
}
```

##### Opção 2 — Redis (produção distribuída)

Se tivermos múltiplas instâncias do backend (horizontal scaling), o rate limiter em memória não funciona porque cada instância tem seu próprio balde. Nesse caso, usamos Redis:

```go
// Usa o comando INCR + EXPIRE do Redis
// Chave: "rate:userID:minuto_atual"
// Se count > limit → rejeita
func (rl *RedisRateLimiter) Allow(ctx context.Context, userID string) bool {
    key := fmt.Sprintf("rate:%s:%d", userID, time.Now().Unix()/60)
    count, _ := rl.redis.Incr(ctx, key).Result()
    if count == 1 {
        rl.redis.Expire(ctx, key, 2*time.Minute)  // Auto-limpa
    }
    return count <= int64(rl.limit)
}
```

##### Limites sugeridos

| Endpoint | Limite sugerido | Motivo |
|---|---|---|
| `POST /vistorias` | 5/minuto | Custa R$103,56 cada — spam improvável |
| `GET /vistorias/:protocol` | 30/minuto | Gratuito, mas não precisa abusar |
| `GET /vistorias/:protocol/relatorio` | 10/minuto | Custa R$30,96 |
| `GET /consultas/veicular/:tipo/:valor` | 20/minuto | Consulta paga |
| `POST /auth/login` | 5/minuto | Prevenção de força bruta |

#### Conceitos relacionados

- **Token Bucket** — algoritmo que permite bursts controlados de tráfego
- **Sliding Window** — variação que conta requests em uma janela deslizante de tempo
- **429 Too Many Requests** — código HTTP padrão para rate limiting
- **API Gateway** — em produção, o rate limiting pode ser feito no Google Cloud API Gateway ou Cloudflare, antes de chegar no backend

---

### 8. Logging Estruturado para Auditoria

#### O que é

Logging estruturado é registrar eventos em formato **parseável** (JSON) em vez de texto livre. Isso permite:

- **Auditoria financeira** — rastrear cada débito e crédito (quem, quando, quanto, por quê)
- **Debug de produção** — entender o que aconteceu sem acessar o banco
- **Alertas automáticos** — configurar alertas quando padrões suspeitos aparecem
- **Compliance** — provar que operações financeiras foram executadas corretamente

#### O que logar

##### Operações financeiras (OBRIGATÓRIO)

Toda operação que altera saldo deve gerar um log:

```json
{
  "level": "info",
  "event": "balance_debit",
  "user_id": "550e8400-...",
  "amount_cents": 10356,
  "product": "vistoria_digital",
  "protocol": "18de85c0",
  "balance_before": 50000,
  "balance_after": 39644,
  "timestamp": "2026-03-17T15:30:00Z"
}
```

```json
{
  "level": "info",
  "event": "balance_credit_rollback",
  "user_id": "550e8400-...",
  "amount_cents": 10356,
  "reason": "api_call_failed",
  "error": "connection refused",
  "timestamp": "2026-03-17T15:30:01Z"
}
```

```json
{
  "level": "info",
  "event": "payment_confirmed",
  "user_id": "550e8400-...",
  "order_id": "abc123",
  "amount_cents": 5000,
  "provider": "picpay",
  "timestamp": "2026-03-17T15:30:00Z"
}
```

##### Eventos de segurança (RECOMENDADO)

```json
{
  "level": "warn",
  "event": "unauthorized_sync_attempt",
  "user_id": "atacante-id",
  "order_owner": "vitima-id",
  "reference_id": "ref-123",
  "timestamp": "2026-03-17T15:30:00Z"
}
```

```json
{
  "level": "warn",
  "event": "credit_blocked_production",
  "user_id": "550e8400-...",
  "amount_cents": 999999,
  "timestamp": "2026-03-17T15:30:00Z"
}
```

#### Como implementar em Go

##### Opção 1 — `log/slog` (stdlib, Go 1.21+)

```go
import "log/slog"

// No service, após débito
slog.Info("balance_debit",
    "user_id", userID,
    "amount_cents", cost,
    "product", "vistoria_digital",
)

// No rollback
slog.Warn("balance_credit_rollback",
    "user_id", userID,
    "amount_cents", cost,
    "reason", "api_call_failed",
    "error", err.Error(),
)
```

##### Opção 2 — `zerolog` (mais performático)

```go
import "github.com/rs/zerolog/log"

log.Info().
    Str("event", "balance_debit").
    Str("user_id", userID).
    Int64("amount_cents", cost).
    Str("product", "vistoria_digital").
    Msg("saldo debitado")
```

#### Onde os logs vão parar

- **Local**: stdout (visível no terminal)
- **Cloud Run**: Google Cloud Logging (automático, basta logar no stdout em JSON)
- **Análise**: BigQuery ou Loki para queries sobre os logs

#### Conceitos relacionados

- **Structured Logging** — logs em formato parseável (JSON) em vez de texto livre
- **Audit Trail** — registro imutável de todas as operações financeiras
- **Observability** — a capacidade de entender o estado interno do sistema olhando de fora (logs + metrics + traces)
- **ELK Stack / Grafana Loki** — ferramentas para armazenar, buscar e visualizar logs

---

### 9. Testes de Integração com API Real

#### O que é

Testes de integração verificam que o sistema funciona **com as dependências reais** (API da Infovist, Firestore, PicPay), diferente dos testes unitários que usam mocks.

#### Por que são necessários

Os testes unitários garantem que a **lógica interna** está correta, mas não garantem que:

- A API da Infovist aceita o formato dos dados que enviamos
- Os campos da resposta da Infovist batem com nossos structs
- O token de autenticação funciona e renova corretamente
- O Firestore transaction realmente previne race conditions em produção

#### Como organizar

##### Ambiente de staging

Criar um ambiente separado para testes que usa credenciais reais mas isoladas:

```
INFOVIST_EMAIL=staging@buskatotal.com
INFOVIST_PASSWORD=staging-password
INFOVIST_API_TOKEN=staging-token
INFOVIST_BASE_URL=https://api.infovist.com.br/api/v1
FIREBASE_PROJECT_ID=buskatotal-staging
```

##### Estrutura dos testes

```go
// internal/infra/infovist/client_integration_test.go

//go:build integration
// Esse build tag impede que rode com `go test ./...`
// Só roda com: go test -tags=integration ./...

func TestAuthenticate_Real(t *testing.T) {
    client := NewClient(
        os.Getenv("INFOVIST_BASE_URL"),
        os.Getenv("INFOVIST_EMAIL"),
        os.Getenv("INFOVIST_PASSWORD"),
        os.Getenv("INFOVIST_API_TOKEN"),
    )

    resp, err := client.Authenticate(context.Background())
    if err != nil {
        t.Fatalf("auth failed: %v", err)
    }
    if resp.AccessToken == "" {
        t.Fatal("expected non-empty access token")
    }
    t.Logf("token obtained, expires_in: %d", resp.ExpiresIn)
}

func TestCreateInspection_Real(t *testing.T) {
    // ... autenticar primeiro
    // ... criar vistoria com dados de teste
    // ... verificar que o protocolo foi retornado
    // ... consultar status com o protocolo
}
```

##### Como rodar

```bash
# Testes unitários (rápidos, sem dependências externas)
go test ./...

# Testes de integração (lentos, precisa de credenciais)
go test -tags=integration ./internal/infra/infovist/ -v
```

##### CI/CD

No pipeline de CI/CD (GitHub Actions, Cloud Build):

```yaml
# Roda em todo push
- name: Unit Tests
  run: go test ./...

# Roda só antes de deploy para produção
- name: Integration Tests
  run: go test -tags=integration ./...
  env:
    INFOVIST_EMAIL: ${{ secrets.INFOVIST_EMAIL }}
    INFOVIST_PASSWORD: ${{ secrets.INFOVIST_PASSWORD }}
    INFOVIST_API_TOKEN: ${{ secrets.INFOVIST_API_TOKEN }}
```

#### Cuidados

- **Nunca rodar testes de integração em produção** — use credenciais de staging
- **Dados de teste** — usar placas/chassi de teste que não gerem custos reais (combinar com o fornecedor)
- **Limpar dados** — cancelar vistorias criadas em teste para não poluir a conta
- **Rate limiting** — não rodar muitos testes em paralelo para não ser bloqueado pela API

#### Conceitos relacionados

- **Test Pyramid** — muitos testes unitários (base), poucos testes de integração (meio), menos testes E2E (topo)
- **Build Tags** — em Go, permitem incluir/excluir arquivos da compilação com `//go:build tag`
- **Staging Environment** — ambiente que replica produção mas com dados isolados
- **Contract Testing** — verificar que a API do fornecedor retorna o formato esperado

---

## Referências para estudo

- **OWASP Top 10** — lista das 10 vulnerabilidades mais comuns em aplicações web
- **Race Conditions in Go** — https://go.dev/doc/articles/race_detector
- **Firestore Transactions** — operações atômicas no Firestore
- **Saga Pattern** — padrão para transações distribuídas com compensação
- **IDOR** — Insecure Direct Object Reference (OWASP A01:2021)
