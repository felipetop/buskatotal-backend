# Guia de Integração Frontend

## Base URL
- **Local:** `http://localhost:8080`
- **Produção:** `https://buskatotal-api-xxxxxxxxx-rj.a.run.app`

## Autenticação
Todas as rotas protegidas usam JWT no header:
```
Authorization: Bearer <token>
```

Para obter o token:
```
POST /auth/login
{ "email": "user@email.com", "password": "Senha@1234" }
→ { "user": { "id", "name", "email", "balance" }, "token": "eyJ..." }
```

---

## 1. Consultas Infocar (veiculares)

**Rota:** `GET /consultas/veicular/{produto}/{tipo}/{valor}`

**Não precisa de body** — tudo vai na URL.

### Produtos disponíveis

| `produto` | Nome | Preço | `tipo` aceito |
|---|---|---|---|
| `agregados` | Agregados B | R$1,50 | placa, chassi, motor |
| `base-estadual` | Base Estadual B | R$17,70 | placa, chassi |
| `base-nacional` | Base Nacional B | R$18,00 | placa, chassi |
| `gravame` | Gravame B | R$24,00 | placa, chassi |
| `roubo-furto` | Hist. Roubo e Furto B | R$15,60 | placa, chassi |
| `leilao` | Leilão Essencial | R$15,60 | placa, chassi |
| `aquisicoes` | Info Aquisições | R$9,00 | placa |
| `debitos` | Info Débitos | R$6,90 | placa |
| `proprietario` | Hist. Proprietário | R$39,00 | placa, chassi |

### Exemplo de chamada
```typescript
// Angular
this.http.get(`${API}/consultas/veicular/gravame/placa/ABC1234`, {
  headers: { Authorization: `Bearer ${token}` }
}).subscribe(res => {
  // res.solicitacao — metadados
  // res.retorno — status (mensagem: 1 = encontrado)
  // res.dados — dados do veículo (estrutura varia por produto)
});
```

### Response padrão
```json
{
  "solicitacao": { "nomeConsulta", "tipoDado", "dado", ... },
  "retorno": { "mensagem": 1, "descricao": "Consulta realizada com sucesso", ... },
  "dados": { ... }
}
```

### Verificar sucesso
```typescript
if (res.retorno?.mensagem === 1) {
  // Dados encontrados → exibir res.dados
} else if (res.retorno?.mensagem === 0) {
  // Sem registro para essa placa/chassi
} else {
  // Erro (3=dados incorretos, 4=sistema indisponível, 5=limite, 6=auth)
}
```

### Estrutura do `dados` por produto

**agregados:** `dados.dadosDoVeiculoCompleto.{placa, chassi, modelo, cor, ...}`

**base-estadual / base-nacional:**
```
dados.dadosDoVeiculo.{placa, chassi, modelo, ...}
dados.informacoesTecnicasAdcionais.{motor, potencia, ...}
dados.restricoesImpedimentos.{situacaoVeiculo, rouboFurto, restricoes[], intencaoDeGravame}
```

**gravame:** `dados.historico[]` — array de contratos de financiamento

**roubo-furto:**
```
dados.dadosDoVeiculo.{...}
dados.restricoesImpedimentos.{rouboFurto, restricoes[]}
dados.registrosRouboFurto[] — array de ocorrências policiais
```

**leilao:**
```
dados.dadosDoVeiculo.{...}
dados.score — "1"=inteiro, "2"=peq.danos, "3"=méd.danos, "4"=grandes/sucata
dados.registrosLeilao[] — array de registros de leilão
dados.ratingSeguridade.{aceitacaoSeguro, fipeParcial, vistoriaEspecial}
dados.checklistVeiculo — pode ser null
dados.inspecaoVeicular — pode ser null
```

**aquisicoes:** `dados.{placa, chassi, quantidadeAquisicoes, dataUltimaAquisicao, ...}` — campos diretos

**debitos:**
```
dados.dadosDoVeiculo.{totalDebito}
dados.debitosMultas.{totalDebito, quantidade, ocorrencia[]}
dados.debitosIPVA.{totalDebito, quantidade, ocorrencia[]}
dados.debitosDPVAT.{totalDebito, quantidade, ocorrencia[]}
```

**proprietario:** `dados.historicoProprietarioA[]` — array de donos anteriores

---

## 2. Consultas API Full (dados, dívidas, jurídico, veicular)

**Rota:** `POST /consultas/dados/{produto}`

**Precisa de body** com o campo `valor`.

### Produtos disponíveis

#### Dados Cadastrais
| `produto` | Nome | Preço | `valor` |
|---|---|---|---|
| `cpf-simples` | CPF Simples | R$0,30 | CPF (11 dígitos) |
| `cpf-completo` | CPF Completo | R$1,80 | CPF |
| `cpf-ultra` | CPF Ultra Completo | R$3,51 | CPF |
| `busca-nome` | Busca pelo nome | R$4,50 | `NOME\|UF` ex: `JOAO SILVA\|SP` |
| `busca-telefone` | Busca pelo telefone | R$4,50 | `DDD\|NUMERO` ex: `11\|999999999` |
| `cnpj` | CNPJ Completo | R$0,18 | CNPJ (14 dígitos) |

#### Veicular
| `produto` | Nome | Preço | `valor` |
|---|---|---|---|
| `placa-basica` | Placa Básica | R$0,30 | placa |
| `bin-estadual` | BIN Estadual | R$8,28 | placa |
| `bin-nacional` | BIN Nacional | R$9,00 | placa |
| `foto-leilao` | Foto Leilão | R$36,00 | placa |
| `leilao-apifull` | Leilão | R$26,28 | placa |
| `historico-roubo-furto` | Hist. roubo ou furto | R$28,08 | placa |
| `indice-risco` | Índice de risco | R$18,72 | placa |
| `proprietario-placa` | Proprietário placa | R$2,73 | placa |
| `recall` | Recall | R$10,80 | placa |
| `gravame-apifull` | Gravame | R$6,60 | placa |
| `fipe` | Fipe | R$0,33 | placa |
| `csv` | Cert. Segurança Veicular | R$6,60 | placa |
| `crlv` | CRLV | R$60,84 | placa |
| `roubo-furto-apifull` | Hist. roubo e furto | R$10,80 | placa |

#### Dívidas e Crédito
| `produto` | Nome | Preço | `valor` |
|---|---|---|---|
| `spc-srs` | SPC e SRS | R$24,84 | CPF |
| `serasa-premium` | Serasa Premium | R$20,88 | CPF ou CNPJ |
| `cred-completa` | Cred Completa Plus | R$7,47 | CPF ou CNPJ |
| `boavista-essencial` | Boa Vista Essencial | R$9,69 | CPF ou CNPJ |
| `scpc-bv-basica` | SCPC BV Básica | R$7,20 | CPF |
| `cadastrais-score-dividas` | Cadastrais + Score | R$8,28 | CPF ou CNPJ |
| `cadastrais-score-dividas-cp` | Cadastrais + Score CP | R$8,97 | CPF ou CNPJ |
| `scr-bacen` | SCR e Score (BACEN) | R$28,08 | CPF ou CNPJ |
| `cenprot` | CENPROT V2 | R$1,20 | CPF ou CNPJ |
| `quod` | QUOD | R$14,34 | CPF ou CNPJ |

#### Jurídico
| `produto` | Nome | Preço | `valor` |
|---|---|---|---|
| `acoes-processos` | Ações e processos | R$12,42 | CPF ou CNPJ |
| `dossie-juridico` | Dossiê Jurídico | R$35,28 | CPF |
| `cndt` | CNDT | R$21,60 | CPF ou CNPJ |

### Exemplo de chamada
```typescript
// Angular
this.http.post(`${API}/consultas/dados/cpf-simples`,
  { valor: '12345678900' },
  { headers: { Authorization: `Bearer ${token}` } }
).subscribe(res => {
  // res.status — "sucesso" ou erro
  // res.dados — dados retornados
});
```

### Response padrão
```json
{
  "status": "sucesso",
  "dados": { ... }
}
```

### Verificar sucesso
```typescript
if (res.status === 'sucesso' && res.dados) {
  // Exibir res.dados
} else {
  // Erro
}
```

---

## 3. Saldo

**Consultar:**
```
GET /users/{userId}/balance
→ { "user_id", "balance_cents": 2990, "balance_brl": 29.90 }
```

**Verificar antes de consultar:** compare `balance_cents` com o preço do produto.

---

## 4. Diferenças importantes entre Infocar e API Full

| | Infocar | API Full |
|---|---|---|
| **Método** | GET | POST |
| **Dados na** | URL (path params) | Body JSON |
| **Response** | `solicitacao` + `retorno` + `dados` | `status` + `dados` |
| **Verificar sucesso** | `retorno.mensagem === 1` | `status === "sucesso"` |
| **Tipo de consulta** | separado (`tipo` = placa/chassi/motor) | embutido no `valor` |

---

## 5. Tratamento de erros

Todos os endpoints retornam erros no formato:
```json
{ "error": "mensagem do erro" }
```

| HTTP Status | Significado |
|---|---|
| 200 | Sucesso |
| 400 | Erro de validação, saldo insuficiente, ou erro na API do fornecedor |
| 401 | Token inválido ou ausente |
| 404 | Rota não encontrada |

### Saldo insuficiente
Quando o saldo não é suficiente, o erro retornado é:
```json
{ "error": "insufficient balance" }
```
O front deve redirecionar o usuário para a tela de recarga (PicPay).
