// Package http exposes the identity use cases over REST. DTOs here map to/from
// domain types and mirror the frontend's lib/types/models.js — domain aggregates
// are never serialized directly.
package http

import (
	"jago-bahe-backend/internal/identity/application"
	"jago-bahe-backend/internal/identity/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

type registerRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	NID      string `json:"nid"`
	UnionID  string `json:"unionId"`
}

// registerOfficialRequest carries no tier or area: an official claims an office
// that already exists on the directory, so the office's own record supplies both.
// Letting a registrant name their own tier would let them declare themselves MP.
type registerOfficialRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	NID        string `json:"nid"`
	OfficialID string `json:"officialId"`
}

type rejectClaimRequest struct {
	Reason string `json:"reason"`
}

type loginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// userDTO mirrors the frontend User (with officialId for official/admin roles).
type userDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	NID        string `json:"nid,omitempty"`
	UnionID    string `json:"unionId,omitempty"`
	Verified   bool   `json:"verified"`
	OfficialID string `json:"officialId,omitempty"`
}

type loginResponse struct {
	Token string  `json:"token"`
	Role  string  `json:"role"`
	User  userDTO `json:"user"`
}

// officialDTO mirrors the frontend Official.
type officialDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Tier   string `json:"tier"`
	AreaID string `json:"areaId"`
}

func toUserDTO(a *domain.Account) userDTO {
	return userDTO{
		ID:         a.ID,
		Name:       a.Name,
		Phone:      a.Phone.String(),
		Role:       string(a.Role),
		NID:        a.NID,
		UnionID:    a.UnionID.String(),
		Verified:   a.Verified,
		OfficialID: a.OfficialID,
	}
}

func toOfficialDTO(o domain.Official) officialDTO {
	return officialDTO{
		ID:     o.ID,
		Name:   o.Name,
		Phone:  o.Phone.String(),
		Tier:   string(o.Tier),
		AreaID: o.AreaID.String(),
	}
}

type registerOfficialResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

// claimDTO mirrors an official's claim to a directory office.
type claimDTO struct {
	ID         string `json:"id"`
	AccountID  string `json:"accountId"`
	OfficialID string `json:"officialId"`
	Status     string `json:"status"`
	ReviewedBy string `json:"reviewedBy,omitempty"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// pendingClaimDTO is a queue entry: the claim plus who and which office it
// concerns, so a reviewer can judge it without three more requests.
type pendingClaimDTO struct {
	claimDTO
	OfficialName string `json:"officialName"`
	OfficialTier string `json:"officialTier"`
	AreaID       string `json:"areaId"`
	ClaimantName string `json:"claimantName"`
	ClaimantNID  string `json:"claimantNid,omitempty"`
}

func toClaimDTO(c domain.OfficialClaim) claimDTO {
	dto := claimDTO{
		ID:         c.ID,
		AccountID:  c.AccountID,
		OfficialID: c.OfficialID,
		Status:     string(c.Status),
		ReviewedBy: c.ReviewedBy,
		Reason:     c.Reason,
		CreatedAt:  c.CreatedAt.Format(timeLayout),
	}
	if c.ReviewedAt != nil {
		dto.ReviewedAt = c.ReviewedAt.Format(timeLayout)
	}
	return dto
}

// oversightEntryDTO is one moderator decision in the super admin's feed. It is a
// projection of the audit log — the record the platform already keeps — so
// oversight and the public trail can never tell different stories.
type oversightEntryDTO struct {
	ID         string `json:"id"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func toOversightDTO(e auditdomain.AuditEntry) oversightEntryDTO {
	return oversightEntryDTO{
		ID:         e.ID,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
		Actor:      e.Actor,
		Action:     e.Action,
		Reason:     e.Reason,
		CreatedAt:  e.CreatedAt.Format(timeLayout),
	}
}

func toPendingClaimDTO(p application.PendingClaim) pendingClaimDTO {
	return pendingClaimDTO{
		claimDTO:     toClaimDTO(p.Claim),
		OfficialName: p.OfficialName,
		OfficialTier: p.OfficialTier,
		AreaID:       p.AreaID,
		ClaimantName: p.ClaimantName,
		ClaimantNID:  p.ClaimantNID,
	}
}
