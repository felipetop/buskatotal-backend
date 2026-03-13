# BuskaTotal Backend

BuskaTotal é uma **plataforma de consulta veicular, crédito, dados e jurídico**. Este repositório contém o backend em Go (Gin) com Firestore seguindo DDD, DRY e KISS. Inclui CRUD de **Users** e **Tasks**.

## Documentação

- [Documentação técnica](docs/README.md)
- [OpenAPI (Swagger)](docs/openapi.yaml)

## Requisitos

- Go 1.22+ (testado em 1.26.1)
- Conta Firebase + Firestore habilitado
- Arquivo de credenciais (Service Account JSON)

## Configuração

Crie um arquivo de variáveis de ambiente ou exporte no terminal:

```bash
setx FIREBASE_PROJECT_ID "seu-project-id"
setx GOOGLE_APPLICATION_CREDENTIALS "C:\caminho\para\service-account.json"
setx PORT "8080"
setx USE_MOCK_DB "false"
```

> Observação: Após `setx`, abra um novo terminal para recarregar o PATH/envs.

## Rodando a API

```bash
go mod tidy
go run ./cmd/api
```

### Rodar sem Firebase (Mock em memória)

```bash
setx USE_MOCK_DB "true"
go run ./cmd/api
```

> O mock mantém os dados apenas em memória (apaga ao reiniciar).

## Endpoints

### Health
- `GET /health`

### Users
- `POST /users`
- `GET /users`
- `GET /users/:id`
- `PUT /users/:id`
- `DELETE /users/:id`

Exemplo payload:

```json
{
  "name": "Maria",
  "email": "maria@email.com"
}
```

### Tasks
- `POST /tasks`
- `GET /tasks?userId=...`
- `GET /tasks/:id`
- `PUT /tasks/:id`
- `DELETE /tasks/:id`

Exemplo payload:

```json
{
  "userId": "user-id",
  "title": "Comprar pão",
  "description": "Ir à padaria",
  "done": false
}
```

### Infocar (Agregados B)
- `GET /infocar/agregados-b/:tipo/:valor`

Headers obrigatórios:
- `X-User-Id`: ID do usuário com saldo (mock)

Exemplo:

```
GET /infocar/agregados-b/placa/ABC1234
X-User-Id: <ID_DO_USUARIO>
```

## Estrutura (DDD)

```
cmd/api            -> bootstrap do servidor
configs            -> configuração e envs
internal/domain    -> entidades e interfaces de repositório
internal/app       -> serviços (casos de uso)
internal/infra     -> Firestore (implementações)
internal/interfaces/http -> handlers e rotas HTTP
```
