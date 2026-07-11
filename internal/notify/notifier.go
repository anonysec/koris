package notify

// Notifier provides unified notification handling across email and Telegram.
type Notifier struct {
	email *EmailSender
	tg    *TelegramSender
}

// NewNotifier creates a new notifier from environment configuration.
func NewNotifier() *Notifier {
	return &Notifier{
		email: NewEmailSender(LoadEmailConfig()),
		tg:    NewTelegramSender(LoadTelegramConfig()),
	}
}

// SendEmail sends an email notification.
func (n *Notifier) SendEmail(to, subject, body string) error {
	return n.email.Send(to, subject, body)
}

// SendTelegram sends a Telegram message.
func (n *Notifier) SendTelegram(message string) error {
	if n.tg == nil || !n.tg.IsEnabled() {
		return nil
	}
	n.tg.Send(message)
	return nil
}

// Notify sends notifications through all configured channels.
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

// Send forwards a raw message to the configured Telegram chat.
func (n *Notifier) Send(message string) {
	if n.tg != nil {
		n.tg.Send(message)
	}
}

// SendEvent forwards a structured event notification to Telegram.
func (n *Notifier) SendEvent(eventType, title, detail string) {
	if n.tg != nil {
		n.tg.SendEvent(eventType, title, detail)
	}
}

// NotifyPayment forwards a payment notification to Telegram.
func (n *Notifier) NotifyPayment(username string, amount float64, status string) {
	if n.tg != nil {
		n.tg.NotifyPayment(username, amount, status)
	}
}

// NotifyCustomerCreated forwards a new-customer notification to Telegram.
func (n *Notifier) NotifyCustomerCreated(username, creator string) {
	if n.tg != nil {
		n.tg.NotifyCustomerCreated(username, creator)
	}
}

// NotifyExpiry forwards a subscription-expiry notification to Telegram.
func (n *Notifier) NotifyExpiry(username string) {
	if n.tg != nil {
		n.tg.NotifyExpiry(username)
	}
}

// NotifyNodeOffline forwards a node-offline notification to Telegram.
func (n *Notifier) NotifyNodeOffline(nodeName, nodeIP string) {
	if n.tg != nil {
		n.tg.NotifyNodeOffline(nodeName, nodeIP)
	}
}

// NotifyNodeOnline forwards a node-online notification to Telegram.
func (n *Notifier) NotifyNodeOnline(nodeName, nodeIP string) {
	if n.tg != nil {
		n.tg.NotifyNodeOnline(nodeName, nodeIP)
	}
}

// IsEnabled reports whether the Telegram channel is configured.
func (n *Notifier) IsEnabled() bool {
	return n.tg != nil && n.tg.IsEnabled()
}
