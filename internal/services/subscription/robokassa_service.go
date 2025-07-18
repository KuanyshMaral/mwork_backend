package subscription

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type RobokassaService struct {
	MerchantLogin string
	Password1     string
	Password2     string
	BaseURL       string
	Currency      string
}

// NewRobokassaService инициализирует сервис на основе переменных окружения или конфигов.
func NewRobokassaService() *RobokassaService {
	return &RobokassaService{
		MerchantLogin: os.Getenv("ROBOKASSA_LOGIN"),
		Password1:     os.Getenv("ROBOKASSA_PASSWORD1"),
		Password2:     os.Getenv("ROBOKASSA_PASSWORD2"),
		BaseURL:       "https://auth.robokassa.kz/Merchant/Index.aspx", // или .ru
		Currency:      "KZT",                                           // можно параметризовать
	}
}

// GeneratePaymentURL создаёт ссылку на оплату.
func (r *RobokassaService) GeneratePaymentURL(orderID string, amount float64, description, email string) (string, error) {
	signature := r.generateSignature(orderID, amount)
	params := url.Values{}

	params.Set("MrchLogin", r.MerchantLogin)
	params.Set("OutSum", fmt.Sprintf("%.2f", amount))
	params.Set("InvId", orderID)
	params.Set("Desc", description)
	params.Set("SignatureValue", signature)
	params.Set("Email", email)
	params.Set("IncCurrLabel", r.Currency)
	params.Set("Culture", "ru")

	return fmt.Sprintf("%s?%s", r.BaseURL, params.Encode()), nil
}

// generateSignature формирует MD5-подпись для оплаты.
func (r *RobokassaService) generateSignature(orderID string, amount float64) string {
	plain := fmt.Sprintf("%s:%.2f:%s:%s", r.MerchantLogin, amount, orderID, r.Password1)
	hash := md5.Sum([]byte(plain))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// VerifyResultSignature проверяет подпись от Robokassa (используется при callback'ах).
func (r *RobokassaService) VerifyResultSignature(amount float64, orderID, receivedSig string) bool {
	expected := fmt.Sprintf("%.2f:%s:%s", amount, orderID, r.Password2)
	hash := md5.Sum([]byte(expected))
	expectedSig := strings.ToUpper(hex.EncodeToString(hash[:]))
	return strings.EqualFold(expectedSig, receivedSig)
}
