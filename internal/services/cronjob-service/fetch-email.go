package cronjobservice

import (
	"fmt"
	"prime-customer-care/internal/services/cronjob"
)

func init() {
	cronjob.RegisterJob("FetchEmail", FetchEmail, "*/1 * * * *")
}

func FetchEmail() {
	fmt.Println("Fetch Email")
}
