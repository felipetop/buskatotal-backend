
# Infocar API – Integração REST
Documentação simplificada para desenvolvedores

Baseado no manual: **Web Services - RESTFul – Karin Belan Rodrigues (Infocar)**

---

# 📌 Visão Geral

A API da **Infocar** fornece dados veiculares através de **REST APIs em JSON**.

O fluxo de integração possui **4 etapas principais**:

1. Gerar chave Base64 (Basic Auth)
2. Gerar Token JWT
3. Preparar requisição autenticada
4. Consumir os endpoints de consulta

---

# 🔐 Credenciais

A autenticação utiliza:

- `infocar-id-key`
- `usuario`
- `senha`

Esses dados são fornecidos pela Infocar no momento da contratação.

⚠️ **Nunca exponha essas credenciais em código público.**

---

# 1️⃣ Gerar chave Base64

Combine:

```
usuario:senha
```

Exemplo:

```
meuUsuario:minhaSenha
```

Converter para Base64.

### Exemplo em JavaScript

```javascript
const base64 = Buffer
  .from("usuario:senha")
  .toString("base64");

console.log(base64);
```

Resultado:

```
dXN1YXJpbzpzZW5oYQ==
```

Guarde essa chave.

---

# 2️⃣ Gerar TOKEN (JWT)

Endpoint:

```
POST https://api.datacast3.com/api/Token/GerarToken
```

Body:

```json
{
  "chave": "SUA_CHAVE_BASE64"
}
```

### Exemplo

```javascript
fetch("https://api.datacast3.com/api/Token/GerarToken", {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    chave: base64
  })
})
.then(res => res.json())
.then(data => console.log(data));
```

Resposta:

```
{
  "token": "JWT_TOKEN"
}
```

⚠️ O token tem validade de **8 horas**.

---

# 3️⃣ Preparar requisição

Todas as chamadas devem incluir:

### Headers

```
infocar-id-Key: SEU_ID
Authorization: Bearer SEU_TOKEN
```

---

# 4️⃣ Consumir os endpoints

Base da API:

```
https://api.datacast3.com/api
```

---

# 📊 Principais Rotas

## Agregados

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/AgregadosB/placa/{placa} |
| Por chassi | /api/v1.0/AgregadosB/chassi/{chassi} |
| Por motor | /api/v1.0/AgregadosB/motor/{motor} |

---

## Leilão

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/LeilaoEssencial/placa/{placa} |
| Por chassi | /api/v1.0/LeilaoEssencial/chassi/{chassi} |

---

## Base Estadual

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/BaseEstadualB/placa/{placa} |
| Por chassi | /api/v1.0/BaseEstadualB/chassi/{chassi} |

---

## Base Nacional

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/BaseNacionalB/placa/{placa} |
| Por chassi | /api/v1.0/BaseNacionalB/chassi/{chassi} |

---

## Gravame

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/GravameB/placa/{placa} |
| Por chassi | /api/v1.0/GravameB/chassi/{chassi} |

---

## Débitos

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v4.0/InfoDebitos/placa/{placa} |

---

## Roubo e Furto

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/HistoricoRouboFurtoB/placa/{placa} |
| Por chassi | /api/v1.0/HistoricoRouboFurtoB/chassi/{chassi} |

---

## Histórico de Proprietários

| Consulta | Endpoint |
|--------|--------|
| Por placa | /api/v1.0/HistoricoProprietarioA/placa/{placa} |
| Por chassi | /api/v1.0/HistoricoProprietarioA/chassi/{chassi} |

---

# 💻 Exemplo completo de consulta

```javascript
async function consultarDebitos(placa, token) {

  const response = await fetch(
    `https://api.datacast3.com/api/v4.0/InfoDebitos/placa/${placa}`,
    {
      method: "GET",
      headers: {
        "infocar-id-Key": "SEU_ID",
        "Authorization": `Bearer ${token}`
      }
    }
  );

  return response.json();
}
```

---

# 🧱 Arquitetura recomendada

Para projetos modernos:

### Backend

- Node.js
- NestJS
- Spring Boot

### Estrutura

```
services/
   tokenService
repositories/
   infocarRepository
controllers/
   consultaController
```

---

# 🚀 Possíveis aplicações

- Consulta de placa
- Histórico de veículos
- Sistema de vistoria
- Plataforma de checagem veicular
- Marketplace automotivo

---

# 📌 Observações

- Token expira em **8 horas**
- Controle de consumo pode ser feito por **cotas**
- Algumas rotas podem exigir autorização adicional da Infocar

---

# 🧠 Dica

Use **cache de token** para evitar gerar um token novo a cada requisição.

Exemplo:

```
Redis
Memory cache
```

---

# 📄 Licença

Documentação derivada do manual de integração Infocar.
