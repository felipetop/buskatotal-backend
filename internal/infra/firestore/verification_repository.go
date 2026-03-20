package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"buskatotal-backend/internal/domain/verification"
)

type VerificationRepository struct {
	client *firestore.Client
}

func NewVerificationRepository(client *firestore.Client) *VerificationRepository {
	return &VerificationRepository{client: client}
}

func (r *VerificationRepository) collection() *firestore.CollectionRef {
	return r.client.Collection("verification_tokens")
}

func (r *VerificationRepository) Create(ctx context.Context, token verification.Token) (verification.Token, error) {
	now := time.Now()
	token.CreatedAt = now

	doc := r.collection().NewDoc()
	token.ID = doc.ID
	if _, err := doc.Set(ctx, token); err != nil {
		return verification.Token{}, err
	}
	return token, nil
}

func (r *VerificationRepository) GetByToken(ctx context.Context, tokenStr string) (verification.Token, error) {
	query := r.collection().Where("token", "==", tokenStr).Limit(1)
	snaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return verification.Token{}, err
	}
	if len(snaps) == 0 {
		return verification.Token{}, verification.ErrTokenNotFound
	}

	var token verification.Token
	if err := snaps[0].DataTo(&token); err != nil {
		return verification.Token{}, err
	}
	return token, nil
}

func (r *VerificationRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.collection().Doc(id).Update(ctx, []firestore.Update{
		{Path: "used", Value: true},
	})
	return err
}

func (r *VerificationRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := r.collection().Where("userId", "==", userID)
	snaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		if _, err := snap.Ref.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}
