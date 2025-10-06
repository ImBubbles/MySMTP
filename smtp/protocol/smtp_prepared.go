package protocol

import "MySMTP/util"

// Server messaging
var (
	PREPARED_S_ACCEPTANCE         string = NewSMTPBuilder().Code(CODE_READY).Message("Service Ready").Get()
	PREPARED_S_BAD_COMMAND        string = NewSMTPBuilder().Code(CODE_INTERNAL_SERVER_ERROR).Message("Syntax error, command not understood").Get()
	PREPARED_S_BAD_SYNTAX         string = NewSMTPBuilder().Code(CODE_BAD_SYNTAX).Message("Syntax error in parameters or arguments").Get()
	PREPARED_S_BAD_SEQUENCE       string = NewSMTPBuilder().Code(CODE_BAD_SEQUENCE).Message("Bad sequence of commands").Get()
	PREPARED_S_ACKNOWLEDGE        string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("OK").Get()
	PREPARED_S_STARTTLS_READY     string = NewSMTPBuilder().Code(CODE_READY).Message("Ready to start TLS").Get()
	PREPARED_S_TLS_REQUIRED       string = NewSMTPBuilder().Code(CODE_BAD_SEQUENCE).Message("TLS connection required").Get()
	PREPARED_S_AUTH_SUCCESS       string = NewSMTPBuilder().Code(CODE_AUTH_SUCCESS).Message("Auth successful").Get()
	PREPARED_S_AUTH_FAILED        string = NewSMTPBuilder().Code(CODE_AUTH_FAILED).Message("Auth failed").Get()
	PREPARED_S_USERNAME64         string = NewSMTPBuilder().Message(util.String64("Username:")).Get()
	PREPARED_S_PASSWORD64         string = NewSMTPBuilder().Message(util.String64("Password:")).Get()
	PREPARED_S_TRANSACTION_FAILED string = NewSMTPBuilder().Code(CODE_FAILURE).Message("Transaction failed").Get()
	PREPARED_S_RELAY_NOT_ALLOWED  string = NewSMTPBuilder().Code(CODE_FAILURE).Message("Cannot relay on this server").Get()
	PREPARED_S_RELAY_ONLY         string = NewSMTPBuilder().Code(CODE_FAILURE).Message("Relay server").Get()
	PREPARED_S_START_MAIL         string = NewSMTPBuilder().Code(CODE_START_MAIL_INPUT).Message("Start mail input; end with <CRLF>.<CRLF>").Get()
	PREPARED_S_BYE                string = NewSMTPBuilder().Code(CODE_QUIT).Message("Bye").Get()
)

// ADVERTISING
var (
	PREPARED_S_ADVERTISE_TLS        string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("STARTTLS").Get()
	PREPARED_S_ADVERTISE_SIZE       string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("SIZE %s").Get()
	PREPARED_S_ADVERTISE_HELP       string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("HELP").Get()
	PREPARED_S_ADVERTISE_AUTH       string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("AUTH %s").Get()
	PREPARED_S_ADVERTISE_PIPELINING string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("PIPELINING").Get()
	PREPARED_S_ADVERTISE_8BITMIME   string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("8BITMIME").Get()
	PREPARED_S_ADVERTISE_HELLO      string = NewSMTPBuilder().Code(CODE_ACKNOWLEDGE).Message("%s HELLO").Get()
)

// Client messaging
var (
	PREPARED_C_EHLO     string = NewSMTPBuilder().Command(COMMAND_EHLO).Get()
	PREPARED_C_QUIT     string = NewSMTPBuilder().Command(COMMAND_QUIT).Get()
	PREPARED_C_STARTTLS string = NewSMTPBuilder().Command(COMMAND_STARTTLS).Get()
)
