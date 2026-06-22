package notification

import (
	"time"

	"github.com/wneessen/go-mail"
)

const NotificationCooldown = 5 * time.Minute

var NotificationTimer = time.Now()
var MailClient *mail.Client

func NewMailClient(smptServer, smtpUsername, smtpPassword string) (*mail.Client, error) {
	client, err := mail.NewClient(smptServer, mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithUsername(smtpUsername), mail.WithPassword(smtpPassword))
	if err != nil {
		return &mail.Client{}, err
	}

	return client, nil
}

func SendMessage(client *mail.Client, imagePath string) error {
	message := mail.NewMsg()

	message.From("testvms123@outlook.com")
	message.To("fio-11111@yandex.ru")
	message.Subject("Обнаружение на камере")
	message.SetBodyString(mail.TypeTextPlain, "Был обнаружен человек на камере")
	message.AttachFile(imagePath)

	if err := client.DialAndSend(message); err != nil {
		return err
	}

	return nil
}
