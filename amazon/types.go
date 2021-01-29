package amazon

// StatementEntry dictates what this policy allows or doesn't allow.
type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

// PolicyDocument is our definition of our policies to be uploaded to AWS Identity and Access Management (IAM).
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}
