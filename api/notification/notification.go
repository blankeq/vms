package notification

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wneessen/go-mail"
)

const NotificationCooldown = 5 * time.Minute

var MailClient *mail.Client

func NewMailClient(ctx context.Context, smptServer, smtpUsername, smtpPassword string) (*mail.Client, error) {
	client, err := mail.NewClient(smptServer, mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithUsername(smtpUsername), mail.WithPassword(smtpPassword))
	if err != nil {
		return &mail.Client{}, err
	}

	if err := client.DialWithContext(ctx); err != nil {
		return &mail.Client{}, err
	}

	return client, nil
}

func SendMessage(client *mail.Client, imagePath string, cameraId int) error {
	message := mail.NewMsg()

	if err := message.From("artemartemartem04@gmail.com"); err != nil {
		log.Panic(err)
	}

	if err := message.To("fio-11111@yandex.ru"); err != nil {
		log.Panic(err)
	}

	message.Subject(fmt.Sprintf("[Камера %d] Обнаружение", cameraId))

	message.SetBodyString(mail.TypeTextPlain, "Был обнаружен человек на камере")
	message.AttachFile(imagePath)

	if err := client.Send(message); err != nil {
		log.Panic(err)
		return err
	}

	return nil
}
