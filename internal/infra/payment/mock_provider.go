package payment

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"time"

	domain "buskatotal-backend/internal/domain/payment"
)

// MockProvider is used in development / tests.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) CreateOrder(_ context.Context, input domain.CreateOrderInput) (domain.OrderResult, error) {
	qrText := fmt.Sprintf("00020101021226mock%s", input.ReferenceID)
	return domain.OrderResult{
		ReferenceID:  input.ReferenceID,
		PaymentURL:   fmt.Sprintf("https://mock.picpay.com/checkout/%s", input.ReferenceID),
		QRCodeText:   qrText,
		QRCodeBase64: mockQRCodeBase64(qrText),
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}, nil
}

func (p *MockProvider) GetOrderStatus(_ context.Context, referenceID string) (domain.OrderStatus, error) {
	_ = referenceID
	return domain.StatusPaid, nil
}

func (p *MockProvider) Credit(_ context.Context, userID string, amount int64) (domain.Receipt, error) {
	reference := fmt.Sprintf("mock-%d", time.Now().UnixNano())
	return domain.Receipt{
		Provider:  "mock",
		Reference: reference,
		Amount:    amount,
	}, nil
}

// mockQRCodeBase64 generates a simple black-and-white checkerboard PNG that
// visually resembles a QR code placeholder. No external library required.
func mockQRCodeBase64(_ string) string {
	const size = 200
	const cellSize = 10

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	accent := color.RGBA{R: 30, G: 30, B: 30, A: 255}

	// Fill white background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, white)
		}
	}

	// Draw checkerboard cells
	for row := 0; row < size/cellSize; row++ {
		for col := 0; col < size/cellSize; col++ {
			if (row+col)%2 == 0 {
				c := black
				// Corners get a stronger accent to look like QR finder patterns
				if (row < 3 && col < 3) || (row < 3 && col >= size/cellSize-3) || (row >= size/cellSize-3 && col < 3) {
					c = accent
				}
				for dy := 0; dy < cellSize; dy++ {
					for dx := 0; dx < cellSize; dx++ {
						img.SetRGBA(col*cellSize+dx, row*cellSize+dy, c)
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
