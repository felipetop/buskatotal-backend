
# MANUAL DO PRODUTO
# LEILÃO ESSENCIAL

**ID:** L03.02  
**SKU:** LS  
**Cód. Nota:** L03  
**Modelo:** JSON  

---

# SUMÁRIO

- Introdução
- Propósito
- Escopo
- Definições, Acrônimos e Abreviações
- Conteúdo do Produto
- Informações Técnicas de Dados de Processamento
- Dados do Veículo
- Informações Técnicas e Adicionais
- Dados do Veículo de Leilão
- Registro de Leilão
- Score de Leilão
- Dados do Veículo de Remarketing
- Registro de Remarketing
- Inspeção Veicular
- Checklist
- Rating de Seguridade
- API: Informações Técnicas
- Request
- Exemplo Request
- Exemplo Response
- HTTP Status Code
- Autenticação, Rotas e Acesso

---

# INTRODUÇÃO

## PROPÓSITO

Este documento visa expor especificidades de produto de API. Aqui constam informações que, majoritariamente, estão em formato de dicionário de dados incluindo coleção de nomes, atributos e definições.

## ESCOPO

Este documento fornece informações complementares ao Manual básico de Integração e à Coleção do Postman.

Para uma melhor experiência, tenha em mãos os seguintes documentos:

1. Manual do produto  
2. Manual básico de integração  
3. Coleção do Postman  

### Manual do produto

Documento com descrição geral do produto **LEILÃO ESSENCIAL** aplicável em diversos contextos:

- Informações sobre atributos de cada objeto do produto  
- Descrições de APIs em JSON  
- Análise de impactos  
- Termos utilizados  
- Informações complementares do produto  
- Dados para solicitação da pesquisa  
- Visualização de prévia de conteúdo  
- Exemplos de respostas  

### Manual básico de integração

Manual com informações técnicas para execução da integração com o produto **LEILÃO ESSENCIAL**.

### Coleção do Postman

Uma Coleção do Postman é uma biblioteca organizada de solicitações de API pré-configuradas.

Ela permite navegar e interagir com as APIs sem necessidade de implementação direta no código.

---

# DEFINIÇÕES, ACRÔNIMOS E ABREVIAÇÕES

**CHAVE**  
String codificada em Base64 gerada a partir de usuário e senha.

**COLEÇÃO DO POSTMAN**  
Arquivo JSON utilizado no Postman para visualizar e manipular rotas de API.

**FORMATO**

| Código | Descrição |
|------|------|
| A | Alfanumérico |
| N | Numérico |
| D | Decimal |
| Base64 | Codificação de autenticação |
| GUID | Identificador único |
| UUID | Identificador universal único |

**MENSAGEM JSON**

| Código | Significado |
|------|------|
| 0 | Sem registro |
| 1 | Registro encontrado |
| 3 | Dados incorretos |
| 4 | Erro no sistema |
| 5 | Limite excedido |
| 6 | Falha de autenticação |

---

# CONTEÚDO DO PRODUTO

## INFORMAÇÕES TÉCNICAS DE DADOS DE PROCESSAMENTO

| Campo JSON | Tamanho | Formato | Conteúdo |
|------|------|------|------|
| nomeConsulta | 25 | A | Nome da consulta |
| dado | 21 | A | Informação utilizada na pesquisa |
| horaSolicitacao | 20 | datetime | Data da pesquisa |
| mensagem | 2 | N | Status da consulta |
| numeroResposta | 32 | UUID | Identificador da resposta |
| tempoProcessamento | 12 | D | Tempo de processamento |
| dataRetorno | 20 | datetime | Data da resposta |
| descricao | 255 | A | Descrição |
| descricaoChave | X | Base64 | Chave da requisição |
| ip | 15 | IPv4 | IP do usuário |
| nomeUsuario | X | A | Nome do usuário |
| tipoDado | 6 | A | placa ou chassi |
| versaoConsulta | 5 | A | Versão da consulta |

---

# DADOS DO VEÍCULO

| Campo | Descrição |
|------|------|
| anoFabricacao | Ano de fabricação |
| anoModelo | Ano do modelo |
| chassi | Identificador do chassi |
| combustivel | Tipo de combustível |
| cor | Cor do veículo |
| modelo | Marca e modelo |
| municipio | Município |
| origemEmplacamento | Estado de origem |
| placa | Placa |
| renavam | Registro RENAVAM |
| tipoVeiculo | Categoria do veículo |

---

# INFORMAÇÕES TÉCNICAS E ADICIONAIS

| Campo | Descrição |
|------|------|
| capacidadeDeCarga | Capacidade de carga |
| capacidadeDePassageiros | Número de passageiros |
| carroceria | Tipo de carroceria |
| categoria | Categoria do veículo |
| cmt | Peso máximo rebocado |
| dataAtualizacao | Data de atualização |
| documentoFaturado | Documento faturado |
| especie | Espécie do veículo |
| motor | Identificação do motor |
| numeroCaixaCambio | Caixa de câmbio |
| numeroCarroceria | Número da carroceria |
| numeroCilindradas | Cilindradas |
| numeroEixoTraseiroDiferencial | Eixo diferencial |
| numeroTerceiroEixo | Terceiro eixo |
| pbt | Peso bruto total |
| potencia | Potência |
| procedencia | Nacional ou importado |
| quantidadeDeEixos | Número de eixos |
| situacaoChassi | Situação do chassi |
| tipoDocFaturado | Tipo do documento |
| ufFaturado | UF do documento |

---

# DADOS DO VEÍCULO DE LEILÃO

Campos retornados quando o veículo possui histórico em leilão.

| Campo | Descrição |
|------|------|
| placa | Placa |
| chassi | Chassi |
| anoFabricacao | Ano fabricação |
| anoModelo | Ano modelo |
| modelo | Modelo |
| cor | Cor |
| renavam | RENAVAM |
| segmento | Segmento SENATRAN |
| subSegmento | Subsegmento |
| motor | Motor |
| numeroCaixaCambio | Câmbio |
| numeroCarroceria | Carroceria |
| numeroEixoTraseiroDiferencial | Eixo diferencial |
| quantidadeDeEixos | Quantidade de eixos |

---

# REGISTRO DE LEILÃO

| Campo | Descrição |
|------|------|
| leiloeiro | Nome do leiloeiro |
| comitente | Nome do comitente |
| lote | Número do lote |
| dataLeilao | Data do leilão |
| condicoesVeiculo | Condições do veículo |
| situacaoChassi | Situação do chassi |
| condicoesMotor | Condições do motor |
| condicoesCambio | Condições do câmbio |
| condicoesMecanica | Condições mecânicas |
| observacao | Observações |

---

# SCORE DE LEILÃO

| Valor | Significado |
|------|------|
| 1 | Aparentemente inteiro |
| 2 | Pequenos danos |
| 3 | Médios danos |
| 4 | Grandes danos / sucata / alagado |

---

# INSPEÇÃO VEICULAR

| Campo | Descrição |
|------|------|
| data | Data da inspeção |
| link | Link da inspeção |
| garantia | Indicação de garantia |

---

# CHECKLIST

Campos que descrevem inspeções visuais do veículo:

- portaMalas
- frente
- frenteDireita
- frenteEsquerda
- portaFrenteDireita
- portaFrenteEsquerda
- portaTraseiraDireita
- portaTraseiraEsquerda
- pneuFrenteDireita
- pneuFrenteEsquerda
- pneuTraseiraDireita
- pneuTraseiraEsquerda
- bancosDianteiros
- bancosTraseiros
- teto
- observacoes

---

# API: INFORMAÇÕES TÉCNICAS

Para consumir a API da Infocar em JSON é necessário executar uma requisição HTTP.

---

# REQUEST

| Campo | Descrição |
|------|------|
| /{tipo} | placa ou chassi |
| /{tipo}/{dado} | valor consultado |
| Authorization | JWT Token |
| infocar-id-Key | chave do produto |

---

# EXEMPLO REQUEST

```
GET https://api.datacast3.com/api/v1.0/LeilaoEssencial/placa/VALORPLACA
```

Headers

```
Authorization: Bearer TOKEN
infocar-id-Key: SEU_ID
```

---

# HTTP STATUS CODE

| Código | Significado |
|------|------|
| 200 | Sucesso |
| 400 | Input inválido |
| 401 | Não autorizado |
| 404 | Nenhum dado encontrado |
| 500 | Erro interno |

---

# AUTENTICAÇÃO E ACESSO

Consultar **Manual Básico de Integração**.
