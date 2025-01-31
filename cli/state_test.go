package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/cli/clitest"
	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/provisioner/echo"
	"github.com/coder/coder/v2/provisionersdk/proto"
)

func TestStatePull(t *testing.T) {
	t.Parallel()
	t.Run("File", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		owner := coderdtest.CreateFirstUser(t, client)
		templateAdmin, _ := coderdtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		wantState := []byte("some state")
		version := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{
						State: wantState,
					},
				},
			}},
		})
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		// Need to create workspace as templateAdmin to ensure we can read state.
		workspace := coderdtest.CreateWorkspace(t, templateAdmin, owner.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		statefilePath := filepath.Join(t.TempDir(), "state")
		inv, root := clitest.New(t, "state", "pull", workspace.Name, statefilePath)
		clitest.SetupConfig(t, templateAdmin, root)
		err := inv.Run()
		require.NoError(t, err)
		gotState, err := os.ReadFile(statefilePath)
		require.NoError(t, err)
		require.Equal(t, wantState, gotState)
	})
	t.Run("Stdout", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		owner := coderdtest.CreateFirstUser(t, client)
		templateAdmin, _ := coderdtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		wantState := []byte("some state")
		version := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{
						State: wantState,
					},
				},
			}},
		})
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := coderdtest.CreateWorkspace(t, templateAdmin, owner.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		inv, root := clitest.New(t, "state", "pull", workspace.Name)
		var gotState bytes.Buffer
		inv.Stdout = &gotState
		clitest.SetupConfig(t, templateAdmin, root)
		err := inv.Run()
		require.NoError(t, err)
		require.Equal(t, wantState, bytes.TrimSpace(gotState.Bytes()))
	})
}

func TestStatePush(t *testing.T) {
	t.Parallel()
	t.Run("File", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		owner := coderdtest.CreateFirstUser(t, client)
		templateAdmin, _ := coderdtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := coderdtest.CreateWorkspace(t, templateAdmin, owner.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		stateFile, err := os.CreateTemp(t.TempDir(), "")
		require.NoError(t, err)
		wantState := []byte("some magic state")
		_, err = stateFile.Write(wantState)
		require.NoError(t, err)
		err = stateFile.Close()
		require.NoError(t, err)
		inv, root := clitest.New(t, "state", "push", workspace.Name, stateFile.Name())
		clitest.SetupConfig(t, templateAdmin, root)
		err = inv.Run()
		require.NoError(t, err)
	})

	t.Run("Stdin", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		owner := coderdtest.CreateFirstUser(t, client)
		templateAdmin, _ := coderdtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := coderdtest.CreateWorkspace(t, templateAdmin, owner.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		inv, root := clitest.New(t, "state", "push", "--build", strconv.Itoa(int(workspace.LatestBuild.BuildNumber)), workspace.Name, "-")
		clitest.SetupConfig(t, templateAdmin, root)
		inv.Stdin = strings.NewReader("some magic state")
		err := inv.Run()
		require.NoError(t, err)
	})
}
