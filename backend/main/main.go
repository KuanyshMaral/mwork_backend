// @title           mwork API
// @version         1.0
// @description     API для управления модельной компанией (документация Swagger).
// @termsOfService  http://your-terms.com/terms/
// @contact.name    Модельное агенство
// @contact.url     http://contact.company.com
// @contact.email   support@company.com
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @host            localhost:4000
// @BasePath        /

package main

import (
	"mwork_front_fn/backend/app"
	_ "mwork_front_fn/backend/docs"
)

func main() {
	app.Run()
}
