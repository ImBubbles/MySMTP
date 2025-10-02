package protocol

type SMTPCode int16

const (
	CODE_OK                    SMTPCode = 200
	CODE_READY                 SMTPCode = 220
	CODE_QUIT                  SMTPCode = 221
	CODE_AUTH_SUCCESS          SMTPCode = 235
	CODE_ACKNOWLEDGE           SMTPCode = 250
	CODE_START_MAIL_INPUT      SMTPCode = 354
	CODE_NOT_FOUND             SMTPCode = 404
	CODE_UNAVAILABLE           SMTPCode = 421
	CODE_INTERNAL_SERVER_ERROR SMTPCode = 500
	CODE_BAD_SYNTAX            SMTPCode = 501
	CODE_BAD_SEQUENCE          SMTPCode = 503
	CODE_AUTH_FAILED           SMTPCode = 535
	CODE_FAILURE               SMTPCode = 554
)

type SMTPCommands string

const (
	COMMAND_EHLO      SMTPCommands = "EHLO"
	COMMAND_MAIL      SMTPCommands = "MAIL"
	COMMAND_MAIL_FROM SMTPCommands = "MAIL FROM"
	COMMAND_RCPT      SMTPCommands = "RCPT"
	COMMAND_RCPT_TO   SMTPCommands = "RCPT TO"
	COMMAND_DATA      SMTPCommands = "DATA"
	COMMAND_QUIT      SMTPCommands = "QUIT"
	COMMAND_RSET      SMTPCommands = "RSET"
	COMMAND_AUTH      SMTPCommands = "AUTH"
	COMMAND_STARTTLS  SMTPCommands = "STARTTLS"
)

type SMTPBody string

const (
	BODY_7BIT     SMTPBody = "7BIT"
	BODY_8BITMIME SMTPBody = "8BITMIME"
)

type SMTPNotify string

const (
	NOTIFY_SUCCESS SMTPNotify = "SUCCESS"
	NOTIFY_FAILURE SMTPNotify = "FAILURE"
	NOTIFY_DELAY   SMTPNotify = "DELAY"
)

type SMTPOrcpt string

const (
	ORCPT_RFC822 SMTPOrcpt = "RFC822"
	ORCPT_X_TEXT SMTPOrcpt = "X_TEXT"
)

type SMTPStates int

const (
	STATE_DEAD      SMTPStates = iota
	STATE_EHLO      SMTPStates = 1
	STATE_AUTH      SMTPStates = 1
	STATE_MAIL_FROM SMTPStates = 2
	STATE_RCPT_TO   SMTPStates = 3
	STATE_DATA      SMTPStates = 4
)
