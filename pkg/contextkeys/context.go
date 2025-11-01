package contextkeys

// Используем кастомный тип, чтобы избежать коллизий
type contextKey string

// DBContextKey - это ключ, по которому мы будем хранить *gorm.DB в context
const DBContextKey = contextKey("db")
