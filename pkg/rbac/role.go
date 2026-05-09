package rbac

type Role string

const (
	RoleUser        Role = "user"
	RoleContributor Role = "contributor"
	RoleModerator   Role = "moderator"
	RoleAdmin       Role = "admin"
)

func (r Role) Level() int {
    switch r {
    case RoleUser:        return 1
    case RoleContributor: return 2
    case RoleModerator:   return 3
    case RoleAdmin:       return 4
    default:              return 0
    }
}

func (r Role) AtLeast(min Role) bool {
    return r.Level() >= min.Level()
}