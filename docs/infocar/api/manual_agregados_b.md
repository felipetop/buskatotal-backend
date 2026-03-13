
# MANUAL DO PRODUTO
## AGREGADOS B

**Versão:** 1.1  
**ID:** V39.02  
**SKU:** AG  
**Cód. Nota:** V39  
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
- API: Informações Técnicas  
- JSON  
- Request  
- Exemplo Request  
- Exemplo Response  
- HTTP Status Code  
- Autenticação, Rotas e Acesso  
- Canais de Suporte e Atendimento  

---

# HISTÓRICO DE VERSÕES

| Versão | Data | Descrição da Mudança | Responsável |
|------|------|------|------|
| 1.0 | 16/07/2024 | Versão inicial do manual | Caroline Oliveira |
| 1.1 | 27/02/2026 | Revisão geral e atualização | Caroline Oliveira |

**Data da última atualização:**  
27 de fevereiro de 2026  

© 2025 Infocar Tecnologia Ltda.  
Todos os direitos reservados.

INFOCAR  
Rua Presidente Prudente, 69 Centro  
Guarulhos - SP  
CEP 07110-140

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

Documento com descrição geral do produto **AGREGADOS B** aplicável em diversos contextos:

- Informações sobre atributos de cada objeto do produto  
- Descrições de APIs em JSON  
- Análise de impactos  
- Termos utilizados  
- Informações complementares  
- Dados para solicitação da pesquisa  
- Visualização prévia de conteúdo  
- Exemplos de respostas  

### Manual básico de integração

Manual com informações técnicas para execução da integração com o produto **AGREGADOS B**.

### Coleção do Postman

Uma Coleção do Postman é uma biblioteca organizada de solicitações de API pré-configuradas.

Ela permite:

- Navegar pelas APIs
- Compreender rotas
- Testar chamadas sem implementação de código

O uso requer a instalação do **Postman**, pois o arquivo é estruturado para uso nessa ferramenta.

---

# DEFINIÇÕES, ACRÔNIMOS E ABREVIAÇÕES

**CHAVE**  
String codificada em Base64 gerada a partir de usuário e senha.

**COLEÇÃO DO POSTMAN**  
Arquivo JSON utilizado no Postman para visualizar e manipular rotas de API.

**MANUAL DO PRODUTO**  
Documento que cataloga termos importantes relacionados às entidades do produto.

**FORMATO**

| Código | Descrição |
|------|------|
| A | Alfanumérico |
| Base64 | Codificação de usuário e senha |
| D | Decimal |
| yyyy-MM-dd HH:mm:ss.fff | Formato de data e hora |
| N | Numérico |
| NR | Número |
| ss.fff | Segundos com milissegundos |
| GUID | Globally Unique Identifier |
| ID | Identificador único |

**ID ALFANUMÉRICO**  
Identificador exclusivo com letras e números.

**LIMITE**  
Número máximo de consultas permitidas.

**MENSAGEM JSON**

| Código | Significado |
|------|------|
| 0 | Sem registro |
| 1 | Registro encontrado |
| 3 | Dados incorretos |
| 4 | Erro na pesquisa |
| 5 | Limite excedido |
| 6 | Falha de autenticação |

**TAMANHO**  
Quantidade de caracteres.

**UUID**  
Universally Unique Identifier utilizado para garantir unicidade das consultas.

---

# CONTEÚDO DO PRODUTO

## INTRODUÇÃO DE PRODUTO

Este produto fornece os **dados de emplacamento do veículo**.

---

# INFORMAÇÕES TÉCNICAS DE DADOS DE PROCESSAMENTO

| JSON | Tamanho | Formato | Conteúdo |
|------|------|------|------|
| nomeConsulta | 25 | A | Nome da consulta |
| dado | 21 | A | Informação usada na pesquisa |
| horaSolicitacao | 20 | yyyy-MM-dd HH:mm:ss.fff | Data da pesquisa |
| mensagem | 2 | N | Status da consulta |
| numeroResposta | 32 | UUID | Identificador da pesquisa |
| tempoProcessamento | 12 | D | Tempo de processamento |
| dataRetorno | 20 | yyyy-MM-dd HH:mm:ss.fff | Data final da pesquisa |
| descricao | 255 | A | Descrição da mensagem |
| descricaoChave | X | Base64 | Chave usada na requisição |
| ip | 15 | IPv4 | IP do solicitante |
| nomeUsuario | X | A | Nome do usuário |
| tipoDado | 6 | A | placa ou chassi |
| versaoConsulta | 5 | A | Versão da consulta |

---

# DADOS DO VEÍCULO

| Campo | Descrição |
|------|------|
| anoFabricacao | Ano de fabricação |
| anoModelo | Ano do modelo |
| numeroCaixaCambio | Identificador da caixa de câmbio |
| capacidadeDeCarga | Peso máximo suportado |
| numeroCarroceria | Identificador da carroceria |
| chassi | Identificação do chassi |
| numeroCilindradas | Cilindradas do motor |
| cmt | Peso máximo rebocado |
| combustivel | Tipo de combustível |
| cor | Cor do veículo |
| numeroEixoTraseiroDiferencial | Identificador do eixo diferencial |
| modelo | Marca e modelo |
| municipio | Município do veículo |
| motor | Identificação do motor |
| pbt | Peso bruto total |
| placa | Placa do veículo |
| potencia | Potência do motor |
| procedencia | Nacional ou importado |
| quantidadeDeEixos | Número de eixos |
| capacidadeDePassageiros | Número de passageiros |
| situacaoChassi | Estado do chassi |
| carroceria | Tipo de carroceria |
| tipoVeiculo | Categoria do veículo |
| tipoMontagem | Método de montagem |
| uf | Estado de registro |

---

# API: INFORMAÇÕES TÉCNICAS

Para conectar às APIs da **Infocar** em JSON é necessário executar uma requisição HTTP.

---

# REQUEST

| Campo | Conteúdo |
|------|------|
| Rota /{tipo} | placa, motor ou chassi |
| Rota {tipo}/{dado} | valor da consulta |
| Header Authorization | Token JWT |
| Header infocar-id-Key | Identificador do produto |

---

# EXEMPLO REQUEST

```
GET https://api.datacast3.com/api/v1.0/AgregadosB/placa/{valorPlaca}

Headers

infocar-id-Key: "seuID"
Authorization: "Bearer seuTokenJWT"
```

---

# EXEMPLO RESPONSE

```json
{
 "solicitacao": {
   "horaSolicitacao": "2025-23-02 12:57:07.305816",
   "ip": "000.00.000.000",
   "nomeConsulta": "Agregados B",
   "tipoDado": "PLACA",
   "dado": "AAA1111",
   "nomeUsuario": "USUARIO TESTE",
   "descricaoChave": "CHAVE TESTE",
   "versaoConsulta": "V1"
 },
 "retorno": {
   "numeroResposta": "UUID",
   "mensagem": 1,
   "tempoProcessamento": 0.2816665,
   "descricao": "Consulta realizada com sucesso",
   "dataRetorno": "2025-08-04 12:57:07.587483"
 }
}
```

---

# HTTP STATUS CODE

| Código | Descrição |
|------|------|
| 200 | Sucesso |
| 400 | Input inválido |
| 401 | Não autorizado |
| 404 | Nenhum dado encontrado |
| 500 | Erro interno |

---

# AUTENTICAÇÃO, ROTAS E ACESSO

Consultar **Manual Básico de Integração**.

---

# SUPORTE

**Email:** atendimento@infocar.com.br  
**WhatsApp:** (11) 94082-7479  

**Horário:**  
Segunda a sexta, das 08h às 17h
