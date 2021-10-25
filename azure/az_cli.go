package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

type AzCli struct {
	HomeDirectory string
}

func (cli *AzCli) DeploySubscriptionTemplate(name, path string) (string, error) {
	l.Log.Infof("running [az deployment sub create]")

	// set a timeout of 5 minutes for the subshell.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"az", "deployment", "sub", "create",
		"--location=WestUS",
		fmt.Sprintf("--name=%v", name),
		fmt.Sprintf("--template-file=%v", path))

	cmd.Env = cli.generatedEnv()

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run [az deployment sub create] command: %v", err)
	}

	l.Log.Infof("running [az deployment sub show]")
	cmd = exec.CommandContext(ctx, "az", "deployment", "sub", "show", fmt.Sprintf("-n=%v", name))
	cmd.Env = cli.generatedEnv()

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run [az deployment sub show] command: %v", err)
	}

	l.Log.Infof("unmarshaling output into struct")
	azDeploy := AzureDeployment{}
	err = json.Unmarshal(output, &azDeploy)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal subscription output: %v", err)
	}

	l.Log.Infof("parsing uuid from susbcription id")
	subID := getUuidFromSubscription(azDeploy.ID)

	if subID == "" {
		return "", fmt.Errorf("failed to parse uuid from subscription ID: %v", azDeploy.ID)
	}

	return subID, nil
}

func (cli *AzCli) DeleteSubscription(name string) error {
	// set a timeout of 5 minutes for the subshell.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	l.Log.Infof("running [az deployment sub delete]")
	cmd := exec.CommandContext(ctx,
		"az", "deployment", "sub", "delete",
		fmt.Sprintf("--name=%v", name),
	)

	cmd.Env = cli.generatedEnv()

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run [az deployment sub delete] command: %v", err)
	}

	return nil
}

func (cli *AzCli) Login(username, password string) error {
	l.Log.Infof("running [az login]")
	cmd := exec.Command("az", "login", fmt.Sprintf("--username=%v", username), fmt.Sprintf("--password=%v", password))
	cmd.Env = cli.generatedEnv()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to login with az credentials: %v", err.Error())
	}

	return nil
}

func (cli *AzCli) Logout() error {
	l.Log.Infof("running [az logout]")
	cmd := exec.Command("az", "logout")
	cmd.Env = cli.generatedEnv()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to logout az cli: %v", err.Error())
	}

	return nil
}

func (cli *AzCli) Cleanup() error {
	l.Log.Infof("removing tmpdir [%v]", cli.HomeDirectory)
	return os.RemoveAll(cli.HomeDirectory)
}

func (cli *AzCli) generatedEnv() []string {
	return []string{fmt.Sprintf("HOME=%v", cli.HomeDirectory)}
}

var subscriptionIDRegex = regexp.MustCompile(`/subscriptions/(.*)/providers/`)

func getUuidFromSubscription(sub string) string {
	if !subscriptionIDRegex.MatchString(sub) {
		return ""
	}

	matches := subscriptionIDRegex.FindAllStringSubmatch(sub, 1)
	return matches[1][0]
}
