package version

var (
	Branch    = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func Info() string {
	return Branch + "@" + Commit
}
