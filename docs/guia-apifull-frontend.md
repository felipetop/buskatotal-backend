# Nova rota: Consultas de Dados

## Rota
```
POST /consultas/dados/{produto}
Authorization: Bearer <token>
Content-Type: application/json

{ "valor": "12345678900" }
```

## Diferença das consultas veiculares (Infocar)
- Infocar: `GET` com dados na URL
- Esta nova: `POST` com `{ "valor": "..." }` no body

## Response
```json
{
  "status": "sucesso",
  "dados": { ... }
}
```

Verificar sucesso: `res.status === "sucesso"`

Erro: `{ "error": "mensagem" }` (HTTP 400)

---

## Produtos disponíveis

### Dados Cadastrais
| `produto` | Nome | Preço | O que manda no `valor` |
|---|---|---|---|
| `cpf-simples` | CPF Simples | R$0,30 | CPF — `12345678900` |
| `cpf-completo` | CPF Completo | R$1,80 | CPF — `12345678900` |
| `cpf-ultra` | CPF Ultra Completo | R$3,51 | CPF — `12345678900` |
| `busca-nome` | Busca pelo nome | R$4,50 | Nome e UF — `JOAO SILVA\|SP` |
| `busca-telefone` | Busca pelo telefone | R$4,50 | DDD e número — `11\|999999999` |
| `cnpj` | CNPJ Completo | R$0,18 | CNPJ — `12345678000190` |

### Veicular
| `produto` | Nome | Preço | O que manda no `valor` |
|---|---|---|---|
| `placa-basica` | Placa Básica | R$0,30 | Placa — `ABC1234` |
| `bin-estadual` | BIN Estadual | R$8,28 | Placa |
| `bin-nacional` | BIN Nacional | R$9,00 | Placa |
| `foto-leilao` | Foto Leilão | R$36,00 | Placa |
| `leilao-apifull` | Leilão | R$26,28 | Placa |
| `historico-roubo-furto` | Hist. roubo ou furto | R$28,08 | Placa |
| `indice-risco` | Índice de risco | R$18,72 | Placa |
| `proprietario-placa` | Proprietário placa | R$2,73 | Placa |
| `recall` | Recall | R$10,80 | Placa |
| `gravame-apifull` | Gravame | R$6,60 | Placa |
| `fipe` | Fipe | R$0,33 | Placa |
| `csv` | Cert. Segurança Veicular | R$6,60 | Placa |
| `crlv` | CRLV | R$60,84 | Placa |
| `roubo-furto-apifull` | Hist. roubo e furto | R$10,80 | Placa |

### Dívidas e Crédito
| `produto` | Nome | Preço | O que manda no `valor` |
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

### Jurídico
| `produto` | Nome | Preço | O que manda no `valor` |
|---|---|---|---|
| `acoes-processos` | Ações e processos | R$12,42 | CPF ou CNPJ |
| `dossie-juridico` | Dossiê Jurídico | R$35,28 | CPF |
| `cndt` | CNDT | R$21,60 | CPF ou CNPJ |

---

## Exemplo Angular

```typescript
// CPF Simples
this.http.post(`${API}/consultas/dados/cpf-simples`,
  { valor: '12345678900' },
  { headers: { Authorization: `Bearer ${token}` } }
).subscribe(res => {
  if (res.status === 'sucesso') {
    console.log(res.dados); // dados da pessoa
  }
});

// Placa (Fipe)
this.http.post(`${API}/consultas/dados/fipe`,
  { valor: 'ABC1234' },
  { headers: { Authorization: `Bearer ${token}` } }
);

// Serasa Premium com CNPJ
this.http.post(`${API}/consultas/dados/serasa-premium`,
  { valor: '12345678000190' },
  { headers: { Authorization: `Bearer ${token}` } }
);

// Busca por nome (nome|UF)
this.http.post(`${API}/consultas/dados/busca-nome`,
  { valor: 'JOAO SILVA|SP' },
  { headers: { Authorization: `Bearer ${token}` } }
);

// Busca por telefone (DDD|NUMERO)
this.http.post(`${API}/consultas/dados/busca-telefone`,
  { valor: '11|999999999' },
  { headers: { Authorization: `Bearer ${token}` } }
);
```

## Erros comuns
| Erro | Causa |
|---|---|
| `"insufficient balance"` | Saldo insuficiente — redirecionar pra recarga |
| `"unknown product: xxx"` | Produto não existe — verificar a chave |
| `"campo 'valor' é obrigatório no body"` | Body vazio ou sem campo `valor` |
| `"apifull token missing"` | Token não configurado no servidor (avisar admin) |
