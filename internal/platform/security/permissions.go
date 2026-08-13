package security

import "strings"

// Permission is a single capability a handler may require.
//
// Authorization is DENY BY DEFAULT (99 §7): a route that declares no permission
// serves nobody. The constants below must match the codes seeded in
// db/migrations/0011_reference_data.up.sql — a mismatch is caught by
// TestPermissionConstantsMatchTheSeed.
type Permission string

const (
	PermCustomerRead       Permission = "customer.read"
	PermCustomerWrite      Permission = "customer.write"
	PermCustomerTypeChange Permission = "customer.type.change"
	PermOrganisationManage Permission = "organisation.manage"

	PermCatalogueRead  Permission = "catalogue.read"
	PermCatalogueWrite Permission = "catalogue.write"
	PermScheduleRead   Permission = "schedule.read"
	PermScheduleWrite  Permission = "schedule.write"

	PermPriceRead     Permission = "price.read"
	PermPriceWrite    Permission = "price.write"
	PermPackageManage Permission = "package.manage"

	PermOrderRead     Permission = "order.read"
	PermOrderWrite    Permission = "order.write"
	PermOrderCancel   Permission = "order.cancel"
	PermPaymentVerify Permission = "payment.verify"
	PermPaymentRefund Permission = "payment.refund"
	PermCreditAdjust  Permission = "credit.adjust"

	PermDeliveryRead     Permission = "delivery.read"
	PermDeliveryFulfil   Permission = "delivery.fulfil"
	PermDeliveryReassign Permission = "delivery.reassign"

	PermKitchenRead  Permission = "kitchen.read"
	PermKitchenWrite Permission = "kitchen.write"

	PermReportRead      Permission = "report.read"
	PermReportFinancial Permission = "report.financial"

	// Customer, own-scoped. Holding one of these says "you may act on your
	// own X"; it never says which X is yours — ownership is a separate check
	// in the repository, because IDOR is the top risk here (PROMPT §14).
	PermProfileManage       Permission = "profile.manage"
	PermAddressManage       Permission = "address.manage"
	PermOrderCreate         Permission = "order.create"
	PermOrderViewOwn        Permission = "order.view.own"
	PermOrderCancelOwn      Permission = "order.cancel.own"
	PermPaymentProofUpload  Permission = "payment.proof.upload"
	PermPackageViewOwn      Permission = "package.view.own"
	PermDeliveryViewOwn     Permission = "delivery.view.own"
	PermDeliveryScheduleOwn Permission = "delivery.schedule.own"

	PermSettingsRead  Permission = "settings.read"
	PermSettingsWrite Permission = "settings.write"
	PermUserManage    Permission = "user.manage"
	PermAuditRead     Permission = "audit.read"
)

// AllPermissions is every permission the code knows about, used to assert the
// Go constants and the seeded rows have not drifted apart.
func AllPermissions() []Permission {
	return []Permission{
		PermCustomerRead, PermCustomerWrite, PermCustomerTypeChange, PermOrganisationManage,
		PermCatalogueRead, PermCatalogueWrite, PermScheduleRead, PermScheduleWrite,
		PermPriceRead, PermPriceWrite, PermPackageManage,
		PermOrderRead, PermOrderWrite, PermOrderCancel,
		PermPaymentVerify, PermPaymentRefund, PermCreditAdjust,
		PermDeliveryRead, PermDeliveryFulfil, PermDeliveryReassign,
		PermKitchenRead, PermKitchenWrite,
		PermReportRead, PermReportFinancial,
		PermSettingsRead, PermSettingsWrite, PermUserManage, PermAuditRead,
		PermProfileManage, PermAddressManage, PermOrderCreate, PermOrderViewOwn,
		PermOrderCancelOwn, PermPaymentProofUpload, PermPackageViewOwn,
		PermDeliveryViewOwn, PermDeliveryScheduleOwn,
	}
}

// SubjectType distinguishes a customer session from a staff one. It rides in
// the token so a customer token can never be replayed against a staff route
// even if the role claim were somehow wrong.
type SubjectType string

const (
	SubjectCustomer SubjectType = "customer"
	SubjectStaff    SubjectType = "staff"
)

// Role is a role code, matching the seeded rows (PROMPT §3).
type Role string

const (
	RoleCustomer Role = "customer"
	RoleStaff    Role = "staff"
	RoleFinance  Role = "finance"
	RoleKitchen  Role = "kitchen"
	RoleCourier  Role = "courier"
	RoleAdmin    Role = "admin"
)

// IsStaff reports whether a role belongs in the back office.
func (r Role) IsStaff() bool { return r != RoleCustomer && r != "" }

// RequiresTOTP reports whether 2FA is mandatory for the role (docs/03 Q-16).
// Kitchen and courier are exempt: they work from shared or phone devices on a
// service floor, and their accounts are read-mostly and kitchen-scoped.
func (r Role) RequiresTOTP() bool {
	switch r {
	case RoleAdmin, RoleFinance, RoleStaff:
		return true
	}
	return false
}

// Set is a resolved permission set for one authenticated user.
type Set map[Permission]struct{}

// NewSet builds a set from permission codes loaded for the user's roles.
func NewSet(codes []string) Set {
	s := make(Set, len(codes))
	for _, c := range codes {
		if c = strings.TrimSpace(c); c != "" {
			s[Permission(c)] = struct{}{}
		}
	}
	return s
}

// Has reports whether the set grants a permission. An empty set grants nothing,
// which is the deny-by-default rule expressed in one line.
func (s Set) Has(p Permission) bool {
	if s == nil {
		return false
	}
	_, ok := s[p]
	return ok
}

// HasAny reports whether the set grants at least one of the permissions.
func (s Set) HasAny(ps ...Permission) bool {
	for _, p := range ps {
		if s.Has(p) {
			return true
		}
	}
	return false
}

// Codes returns the permissions as sorted-insertion-free strings, for a token
// claim or a debug endpoint.
func (s Set) Codes() []string {
	out := make([]string, 0, len(s))
	for p := range s {
		out = append(out, string(p))
	}
	return out
}
