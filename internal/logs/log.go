package logs

import "log/slog"

func Trigger(id string) slog.Attr {
	return slog.String("trigger", id)
}

func User(id string) slog.Attr {
	return slog.String("user", id)
}

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func Database(name string) slog.Attr {
	return slog.String("database", name)
}
