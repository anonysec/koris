package notify

// Notifier provides unified notification handling
type Notifier struct {
	email *EmailSender
	tg    *TelegramSender
}

// NewNotifier creates a new notifier
func NewNotifier() *Notifier {
	return &Notifier{
		email: NewEmailSender(LoadEmailConfig()),
		tg:    NewTelegramSender(LoadTelegramConfig()),
	}
}

// SendEmail sends an email notification
func (n *Notifier) SendEmail(to, subject, body string) error {
	return n.email.Send(to, subject, body)
}

// SendTelegram sends a Telegram message
func (n *Notifier) SendTelegram(message string) error {
	if n.tg == nil || !n.tg.IsEnabled() {
		return nil
	}
	return n.tg.Send(message)
}

// Notify sends notifications through all configured channels
func (n *Notifier) Notify(emailTo, subject, body, telegramMsg string) error {
	if emailTo != "" {
		if err := n.SendEmail(emailTo, subject, body); err != nil {
			return err
		}
	}
	if telegramMsg != "" {
		if err := n.SendTelegram(telegramMsg); err != nil {
			return err
		}
	}
	return nil
}
