package security

const (
	profileField  string = "profile"
	profilesField string = "profiles"
	tenantIdField string = "tenantId"
)

type User struct {
	ID       string
	Email    string
	Phone    string
	Name     string
	TenantID string
	Profile  string
	Profiles []string
	PhotoURL string
}

type UserCreate struct {
	ID       string
	Email    string
	Phone    string
	Password string
	Name     string
	TenantID string
	Profile  string
	Profiles []string
	PhotoURL string
}

type UserUpdate struct {
	Email    string
	Phone    string
	Password string
	Name     string
	TenantID string
	Profile  string
	Profiles []string
	PhotoURL string
}
