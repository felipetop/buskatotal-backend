# Infovist

## O que é a Infovist

A **Infovist** é a plataforma de **vistoria veicular digital** da Infocar, empresa brasileira especializada em dados e soluções tecnológicas para o mercado automotivo.

A Infovist permite realizar vistorias veiculares de forma 100% digital, eliminando a necessidade de vistorias presenciais, reduzindo custos e acelerando processos de aprovação.

---

## Como funciona

1. **Criação da vistoria** — A empresa cria uma vistoria via API informando dados do veículo (placa ou chassi) e do cliente (nome e celular).
2. **Envio do link** — O cliente recebe um link por WhatsApp/SMS para realizar a vistoria.
3. **Captura de fotos** — O cliente envia fotos do veículo seguindo os passos definidos pelo perfil da vistoria.
4. **Análise automatizada (IA)** — As fotos são analisadas por inteligência artificial para identificar danos, arranhões e amassados.
5. **Relatório** — Um relatório completo é gerado com o estado do veículo, incluindo PDF e dados de APIs consultadas (FIPE, leilão, roubo/furto, gravame, etc).
6. **Avaliação** — Opcionalmente, um avaliador pode aprovar, reprovar ou aprovar com ressalvas.

---

## Principais funcionalidades

### Vistoria Digital
- Envio de link para o cliente realizar a vistoria remotamente
- Upload de fotos do veículo com geolocalização
- Reconhecimento automático de imagens via IA
- Detecção de danos com categorização (arranhões, amassados, etc)
- Verificação de pintura (painting check)
- Suporte a diferentes perfis de vistoria (carros, máquinas agrícolas, etc)

### Consultas integradas
O relatório da vistoria pode incluir dados de diversas APIs:
- **Precificação** — Tabela FIPE e Molicar
- **Leilão** — Histórico em duas bases diferentes
- **Roubo/Furto** — Consulta em bases Infocar e Fenauto
- **Gravame** — Restrições financeiras
- **Sinistro** — Histórico de sinistros
- **Recall** — Chamados de recall
- **Renajud** — Restrições judiciais
- **Renavam** — Dados do veículo
- **Risco comercial** — Índice de risco
- **Base estadual** — Débitos e situação do veículo
- **Alerta de frota** — Veículos de locadoras
- **Alerta de órgão público** — Alertas governamentais
- **Remarketing** — Histórico de remarketing

### Webhooks
A Infovist notifica a aplicação integrada sobre mudanças de status via webhooks:
- **Vistoria** — OPEN, MAKE_REPORT, AWAITING_RECAPTURE
- **Avaliação** — AWAITING_EVALUATION, ASSESSING, APPROVED, APPROVED_WITH_NOTES, REPROVED
- **Relatório** — CLOSED
- **Relatório PDF** — GENERATED

### Retriagem
Permite solicitar reenvio de fotos específicas quando necessário, gerando novo relatório e avaliação.

---

## Para quem é utilizada

- Seguradoras
- Concessionárias e revendas
- Financeiras e bancos
- Empresas de vistoria
- Gerenciadoras de frota
- Plataformas de compra e venda de veículos

---

## API de integração

A Infovist disponibiliza uma API REST para integração, com os seguintes grupos de endpoints:

- **Autenticação** — Login com email, senha e api_token (JWT)
- **Vistoria** — Criar, visualizar, listar, cancelar, finalizar retriagem, listar perfis
- **Relatório** — Obter relatório PDF (v1 e v2), com ou sem dados de IA
- **Avaliação** — Alterar status da análise automatizada
- **Retriagem** — Solicitar nova coleta de fotos

Ambiente de produção: `https://api.infovist.com.br/api/v1` (v1) e `https://api.infovist.com.br/api/v2` (v2)

A especificação completa da API está em `infovist-api.yaml`.

---

## Segurança e credenciais

As credenciais (`email`, `password`, `api_token`) são fornecidas pela Infovist/Infocar.
**Nunca exponha esses dados em repositórios públicos ou no frontend.**
Recomenda-se manter essas credenciais em variáveis de ambiente e usar o backend para consumo da API.
