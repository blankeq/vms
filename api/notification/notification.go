package notification

import (
	"context"
	"fmt"
	"log"
	"os"
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

	return client, nil
}

func SendMessage(client *mail.Client, imagePath string, cameraId int) error {
	message := mail.NewMsg()

	from := os.Getenv("SMTP_USERNAME")
	if err := message.From(from); err != nil {
		log.Panic(err)
	}

	to := os.Getenv("NOTIFY_TO")

	if err := message.To(to); err != nil {
		log.Panic(err)
	}

	message.Subject(fmt.Sprintf("[Камера %d] Обнаружение", cameraId))

	message.SetBodyString(mail.TypeTextPlain, "Был обнаружен человек на камере")
	message.AttachFile(imagePath)

	if err := client.DialAndSend(message); err != nil {
		log.Panic(err)
		return err
	}

	return nil
}
