package http

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// internalItem stores cost price in cents — never exposed to clients.
type internalItem struct {
	name        string
	description string
	costCents   int64
}

type internalCategory struct {
	key       string
	label     string
	iconColor string
	items     []internalItem
}

// internalCatalog holds cost prices. Only sale prices (after markup) are served.
var internalCatalog = []internalCategory{
	{
		key:       "VEICULAR",
		label:     "Veicular",
		iconColor: "blue",
		items: []internalItem{
			{"AGREGADOS B", "Consolidação técnica básica: marca, modelo, versão, ano, combustível, categoria, UF e situação cadastral", 50},
			{"BASE ESTADUAL B", "Dados do DETRAN estadual: situação administrativa, licenciamento e restrições locais", 590},
			{"BASE NACIONAL B", "Dados consolidados nacionais com histórico entre estados", 600},
			{"BIN Estadual", "Dados cadastrais do veículo no estado de emplacamento", 276},
			{"BIN Nacional", "Dados consolidados nacionais (SENATRAN)", 300},
			{"Busca dados pela placa", "Identificação completa do proprietário, contatos e vínculos associados à placa", 3576},
			{"Certificado de Segurança Veicular", "CSV, infrações nacionais, restrições judiciais, recalls e dados cadastrais do veículo", 220},
			{"CRLV", "Informações do Certificado de Registro e Licenciamento do Veículo", 2028},
			{"Fipe", "Valor médio de mercado conforme Tabela FIPE", 11},
			{"Foto Leilão", "Fotografias do veículo registradas em leilões", 1200},
			{"Gravame", "Existência de financiamento ou alienação fiduciária", 220},
			{"GRAVAME B", "Dados detalhados do contrato de financiamento e instituição credora", 800},
			{"Histórico de roubo e furto", "Verificação de ocorrência policial ativa", 360},
			{"Histórico de roubo ou furto", "Consulta ampliada de ocorrências antigas ou recuperadas", 936},
			{"HISTÓRICO ROUBO E FURTO B", "Registro detalhado com número do BO, local e data", 520},
			{"Índice de risco veicular", "Avaliação completa do histórico com score de risco", 624},
			{"INFO AQUISIÇÕES", "Histórico de transferências e transações comerciais", 300},
			{"INFO DÉBITOS INPUT PLACA", "Multas, IPVA e taxas pendentes vinculadas à placa", 230},
			// Custo combinado: INFOVIST (R$10,32 = 1032) + VISTORIA DIGITAL (R$34,52 = 3452) = R$44,84 = 4484 centavos
			{"VISTORIA DIGITAL COMPLETA", "Vistoria veicular digital com análise de danos por IA, histórico e relatório completo", 4484},
			{"LEILÃO ESSENCIAL", "Registro detalhado de participação em leilão e classificação de danos", 520},
			{"Leilão", "Indicação geral de registro em base de leilões", 876},
			{"Placa Básica", "Marca, modelo, versão, ano e cor do veículo", 10},
			{"Placa Super Básica", "Identificação essencial do veículo", 8},
			{"Produtos por dados do veículo", "Lista de peças ou serviços compatíveis", 8},
			{"Proprietário placa", "Identificação do proprietário atual registrado", 91},
			{"Recall", "Campanhas de recall do fabricante", 360},
			{"HISTÓRICO PROPRIETÁRIO", "Quantidade de donos anteriores e tipo de uso", 1300},
			{"WORKFLOW DE AVALIAÇÃO", "Dossiê completo automatizado com score e alertas", 953},
			{"Vip Car", "Relatório completo combinando múltiplas verificações veiculares", 3120},
		},
	},
	{
		key:       "DIVIDAS",
		label:     "Dívidas e Crédito",
		iconColor: "emerald",
		items: []internalItem{
			{"Boa Vista Essencial Positivo", "Dados cadastrais completos, score de crédito, pendências financeiras, protestos e cheques sem fundos", 323},
			{"CENPROT V2", "Consulta nacional de protestos em cartórios com valor, data e situação", 40},
			{"Cred Completa Plus", "Visão ampla do risco financeiro com histórico de consultas e ocorrências", 249},
			{"Dados cadastrais, score e dívidas", "Informações cadastrais, score e lista de pendências financeiras", 276},
			{"Dados cadastrais, score e dívidas CP", "Versão alternativa com bases complementares de crédito", 299},
			{"QUOD", "Informações de crédito e dívidas registradas na base QUOD", 478},
			{"SCPC BV Básica", "Pendências financeiras e protestos nas bases SCPC/Boa Vista", 240},
			{"SCR e Score (BACEN)", "Dados do Sistema de Crédito do Banco Central (empréstimos e financiamentos)", 936},
			{"SCR e Score (BACEN) V2", "Versão atualizada com dados ampliados do relacionamento bancário", 936},
			{"Serasa Premium", "Relatório completo do Serasa com score, dívidas e protestos", 696},
			{"Serasa Relatório Básico", "Dívidas comerciais e protestos registrados", 540},
			{"SPC Brasil", "Pendências financeiras e ações cíveis na base SPC", 863},
			{"SPC e SRS", "Cadastro e restrições financeiras nas bases SPC e SRS", 828},
			{"SRS Premium", "Relatório completo da base SRS com score e pendências", 708},
			{"SRS Premium V2", "Versão atualizada do relatório SRS Premium", 696},
		},
	},
	{
		key:       "DADOS",
		label:     "Dados Cadastrais",
		iconColor: "violet",
		items: []internalItem{
			{"Busca pelo nome", "Dados pessoais associados ao nome, endereços, contatos e vínculos", 150},
			{"Busca pelo telefone", "Identificação do titular do número e dados associados", 150},
			{"CNPJ Completo", "Dados cadastrais completos da empresa e quadro societário", 6},
			{"CPF Completo", "Dados cadastrais amplos do CPF e contatos vinculados", 60},
			{"CPF Simples", "Consulta básica de validação do CPF", 10},
			{"CPF Ultra Completo", "Consulta avançada com dados de múltiplas bases", 117},
		},
	},
	{
		key:       "JURIDICO",
		label:     "Jurídico",
		iconColor: "amber",
		items: []internalItem{
			{"Ações e processos judiciais", "Consulta nacional de processos judiciais vinculados à pessoa física ou jurídica", 414},
			{"Buscar reputação", "Varredura online de menções públicas e conteúdos relacionados ao nome", 650},
			{"Certidão Nacional de Débitos Trabalhistas", "Débitos inadimplidos perante a Justiça do Trabalho", 720},
			{"Dossiê Jurídico", "Relatório jurídico aprofundado com processos e vínculos patrimoniais", 1176},
			{"Reconhecimento Facial", "Busca reversa por imagem facial para identificação", 228},
		},
	},
	{
		key:       "SOCIAL",
		label:     "Social",
		iconColor: "pink",
		items: []internalItem{
			{"Instagram 100 Curtidas Mundiais", "Entrega automática de aproximadamente 100 curtidas internacionais", 118},
			{"Instagram 100 Seguidores Mundiais", "Adição automática de cerca de 100 seguidores internacionais", 357},
			{"Instagram 1000 Curtidas Mundiais", "Entrega de aproximadamente 1000 curtidas internacionais", 1176},
			{"Instagram 1000 Seguidores Mundiais", "Adição automática de cerca de 1000 seguidores internacionais", 3572},
			{"TikTok 100 Seguidores Brasileiros", "Entrega automática de aproximadamente 100 seguidores brasileiros", 310},
			{"TikTok 1000 Seguidores Brasileiros", "Entrega automática de aproximadamente 1000 seguidores brasileiros", 3104},
		},
	},
}

// Public response types

type CatalogItemResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
}

type CatalogCategoryResponse struct {
	Key       string                `json:"key"`
	Label     string                `json:"label"`
	IconColor string                `json:"iconColor"`
	Count     string                `json:"count"`
	Items     []CatalogItemResponse `json:"items"`
}

type CatalogHandler struct {
	markup float64
}

func NewCatalogHandler(markup float64) *CatalogHandler {
	return &CatalogHandler{markup: markup}
}

func (h *CatalogHandler) GetCatalog(c *gin.Context) {
	result := make([]CatalogCategoryResponse, 0, len(internalCatalog))
	for _, cat := range internalCatalog {
		items := make([]CatalogItemResponse, 0, len(cat.items))
		for _, item := range cat.items {
			saleCents := int64(math.Round(float64(item.costCents) * (1 + h.markup)))
			items = append(items, CatalogItemResponse{
				Name:        item.name,
				Description: item.description,
				Price:       formatBRL(saleCents),
			})
		}
		result = append(result, CatalogCategoryResponse{
			Key:       cat.key,
			Label:     cat.label,
			IconColor: cat.iconColor,
			Count:     fmt.Sprintf("%d consultas", len(cat.items)),
			Items:     items,
		})
	}
	c.JSON(http.StatusOK, result)
}

func formatBRL(cents int64) string {
	s := fmt.Sprintf("%.2f", float64(cents)/100.0)
	return "R$" + strings.Replace(s, ".", ",", 1)
}
