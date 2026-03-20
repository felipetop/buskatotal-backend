# Architecture & Technical Roadmap

> Documento vivo — atualizar sempre que uma decisão arquitetural for tomada ou um item for implementado.

---

## Visão geral

O BuskaTotal Backend segue **Clean Architecture** em 4 camadas:

```
cmd/
└── api/              ← ponto de entrada (main)

internal/
├── domain/           ← entidades e interfaces (sem dependências externas)
├── app/              ← casos de uso / serviços
├── infra/            ← implementações concretas (Firestore, PicPay, JWT...)
└── interfaces/http/  ← handlers HTTP (Gin)

configs/              ← leitura de variáveis de ambiente
docs/                 ← documentação (OpenAPI, ADRs, este arquivo)
```

**Regra de dependência:** cada camada só conhece a camada interna. `infra` implementa interfaces definidas em `domain`. `app` depende apenas de `domain`. Handlers dependem apenas de interfaces definidas em `interfaces/http`.

---

## O que já é escalável

| Componente | Motivo |
|---|---|
| Autenticação JWT | Stateless — múltiplas instâncias sem sessão compartilhada |
| Verificação de e-mail | Resend via HTTP (sem SDK), token 256-bit, expiração 24h |
| Handlers HTTP | Sem estado interno — horizontalmente escalável |
| Firestore | Banco gerenciado pelo GCP, escala automática |
| PicPay | Processamento de pagamento delegado a serviço externo |
| Interfaces por contrato | Fácil troca de implementação sem afetar outras camadas |
| MockProvider | Desenvolvimento e testes sem dependências externas |

---

## Roadmap técnico

Os itens abaixo estão priorizados por impacto e urgência. Nenhum bloqueia o lançamento inicial, mas os de **Alta** prioridade devem ser resolvidos antes de ir para produção com carga real.

---

### 1. FirestoreOrderRepository

**Prioridade:** Alta — resolver antes de produção
**Status:** Pendente

**Problema atual:**
```go
// internal/app/app.go
orderRepo := memory.NewOrderRepository() // ← dados perdidos ao reiniciar
```
O repositório de pedidos de pagamento usa memória mesmo em produção. Ao reiniciar o servidor, todos os pedidos são perdidos, o que impede o webhook do PicPay de encontrar o pedido e creditá-lo.

**O que implementar:**
- Criar `internal/infra/firestore/order_repository.go` implementando `payment.OrderRepository`
- Coleção sugerida: `payment_orders`
- Índices necessários: `referenceId` (único), `userId` (lista por usuário)
- Substituir em `app.go`:
  ```go
  // de:
  orderRepo := memory.NewOrderRepository()
  // para:
  orderRepo := firestore.NewOrderRepository(client)
  ```

**Arquivo de referência:** `internal/infra/firestore/user_repository.go` (seguir o mesmo padrão)

---

### 2. CORS restrito por origem

**Prioridade:** Alta — resolver antes de produção
**Status:** Pendente

**Problema atual:**
```go
// internal/app/app.go
c.Header("Access-Control-Allow-Origin", "*") // qualquer origem aceita
```
Qualquer domínio pode fazer requisições para a API. Em produção, isso deve ser restrito ao domínio do frontend.

**O que implementar:**
- Ler origens permitidas via variável de ambiente `ALLOWED_ORIGINS`
- Substituir o middleware CORS atual:
  ```go
  // configs/config.go — adicionar:
  AllowedOrigins string // ex: "https://buskatotal.com,https://app.buskatotal.com"

  // app.go — substituir o middleware atual por:
  router.Use(corsMiddleware(cfg.AllowedOrigins))
  ```
- Manter `*` apenas quando `ALLOWED_ORIGINS` estiver vazio (ambiente de desenvolvimento)

---

### 3. Rate Limiting

**Prioridade:** Média — implementar após lançamento
**Status:** Pendente

**Problema atual:**
Sem limite de requisições por IP ou por usuário. Exposto a abuso (scraping, força bruta, flood de pedidos).

**O que implementar:**
- Adicionar middleware de rate limit usando `golang.org/x/time/rate` (já está no `go.mod` como dependência transitiva) ou biblioteca dedicada
- Limites sugeridos por rota:
  | Rota | Limite |
  |---|---|
  | `POST /auth/login` | 10 req/min por IP |
  | `POST /auth/register` | 5 req/min por IP |
  | `POST /payments/users/:id/orders` | 5 req/min por usuário |
  | `GET /infocar/*` | conforme plano do usuário |
  | `POST /payments/webhook` | 100 req/min por IP (PicPay) |

---

### 4. Processamento assíncrono do webhook (fila)

**Prioridade:** Baixa — apenas com alto volume de pagamentos simultâneos
**Status:** Pendente

**Problema atual:**
O webhook do PicPay é processado de forma síncrona: recebe a chamada → consulta PicPay API → atualiza banco → credita saldo — tudo na mesma requisição HTTP. Com muitos pagamentos simultâneos, isso pode causar timeout e o PicPay pode retentar o webhook.

**O que implementar:**
- Usar **Google Cloud Pub/Sub** (já no ecossistema GCP do projeto)
- Fluxo com fila:
  ```
  PicPay → POST /webhook
      → publica mensagem no Pub/Sub { referenceId }
      → retorna 200 imediatamente ao PicPay

  Worker (goroutine ou Cloud Run job)
      → consome mensagem do Pub/Sub
      → verifica status PicPay
      → credita saldo
  ```
- Benefício extra: reprocessamento automático em caso de falha

---

### 5. Observabilidade (logs estruturados + tracing)

**Prioridade:** Média — importante para diagnosticar problemas em produção
**Status:** Pendente

**Problema atual:**
Sem logs estruturados. Erros são retornados ao cliente mas não registrados no servidor com contexto suficiente para diagnóstico.

**O que implementar:**
- Adicionar logger estruturado (`log/slog` — nativo no Go 1.21+, sem dependência nova)
- Logar em cada camada:
  - Handler: método + rota + status code + latência
  - Service: operação + userID + erro
  - Infra: nome do provider + operação + erro
- Em produção, integrar com **Google Cloud Logging** (recebe JSON logs automaticamente via stdout no Cloud Run)

---

## Decisões arquiteturais registradas

| Data | Decisão | Motivo |
|---|---|---|
| 2026-03 | Clean Architecture com 4 camadas | Separação de responsabilidades, testabilidade, troca de implementação sem reescrita |
| 2026-03 | Firestore como banco principal | Sem gerenciamento de servidor, integração nativa com GCP, escala automática |
| 2026-03 | JWT stateless (sem refresh token) | Simplicidade para V1 — adicionar refresh token quando necessário |
| 2026-03 | PicPay como único gateway de pagamento | Boa cobertura no Brasil, suporte a cartão e PIX em uma única integração |
| 2026-03 | MockProvider ativado sem PICPAY_TOKEN | Desenvolvimento e testes locais sem depender de credenciais reais |
| 2026-03 | Webhook re-verifica status na API PicPay | Nunca confiar no payload do webhook — segurança contra chamadas forjadas |
| 2026-03 | Saldo em centavos (int64) | Evita erros de ponto flutuante em operações financeiras |
| 2026-03 | Verificação de e-mail via Resend (HTTP) | Domínio já verificado, API simples, sem SDK — HTTP puro com `net/http`. Token 256-bit via `crypto/rand`, expiração 24h |
| 2026-03 | Envio de verificação assíncrono | Falha no envio de e-mail não deve bloquear o registro do usuário |
| 2026-03 | Coleção `verification_tokens` no Firestore | Separação clara de responsabilidades — tokens de verificação não poluem a coleção `users` |
| 2026-03 | Recuperação de senha com token de 1h | Token mais curto que verificação (1h vs 24h) — reset de senha é operação mais sensível |
| 2026-03 | `forgot-password` sempre retorna 200 | Nunca revelar se um e-mail está cadastrado — previne user enumeration |
| 2026-03 | LGPD — `accepted_terms_at` obrigatório no registro | Timestamp gerado pelo backend (não confiar no front). Rejeita cadastro sem aceite |
| 2026-03 | LGPD — endpoints de direitos do titular | `GET /users/:id/data`, `GET /users/:id/data/export`, `POST /users/:id/data/deletion-request` |
| 2026-03 | LGPD — exclusão = anonimização | Dados financeiros mantidos por 5 anos (obrigação fiscal). Dados pessoais anonimizados |
| 2026-03 | Coleções `deletion_requests` e `data_processing_log` | Auditoria LGPD — registro de todas as operações sobre dados pessoais |
