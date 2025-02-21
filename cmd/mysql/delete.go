package mysql

import (
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/databases/mysql"
	"github.com/tinsane/storages/storage"
	"github.com/tinsane/tracelog"
	"github.com/wal-g/wal-g/utility"
)

var confirmed = false

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete", //for example "delete mysql before time"
	Short: "Clears old backups and binlogs",
}

var deleteBeforeCmd = &cobra.Command{
	Use:     "before [FIND_FULL] backup_name|timestamp", // TODO : improve description
	Example: internal.DeleteBeforeExamples,
	Args:    internal.DeleteBeforeArgsValidator,
	Run:     runDeleteBefore,
}

var deleteRetainCmd = &cobra.Command{
	Use:       "retain [FULL|FIND_FULL] backup_count", // TODO : improve description
	Example:   internal.DeleteRetainExamples,
	ValidArgs: internal.StringModifiers,
	Args:      internal.DeleteRetainArgsValidator,
	Run:       runDeleteRetain,
}

func runDeleteBefore(cmd *cobra.Command, args []string) {
	folder, err := internal.ConfigureFolder()
	tracelog.ErrorLogger.FatalOnError(err)
	isFullBackup := func(object storage.Object) bool {
		return IsFullBackup(folder, object)
	}
	internal.HandleDeleteBefore(folder, args, confirmed, isFullBackup, GetLessFunc(folder))
}

func runDeleteRetain(cmd *cobra.Command, args []string) {
	folder, err := internal.ConfigureFolder()
	tracelog.ErrorLogger.FatalOnError(err)
	isFullBackup := func(object storage.Object) bool {
		return IsFullBackup(folder, object)
	}
	internal.HandleDeleteRetain(folder, args, confirmed, isFullBackup, GetLessFunc(folder))
}

func init() {
	Cmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteBeforeCmd, deleteRetainCmd)
	deleteCmd.PersistentFlags().BoolVar(&confirmed, internal.ConfirmFlag, false, "Confirms backup deletion")
}

func IsFullBackup(folder storage.Folder, object storage.Object) bool {
	return true
}

func GetLessFunc(folder storage.Folder) func(object1, object2 storage.Object) bool {
	return func(object1, object2 storage.Object) bool {
		time1, ok := utility.TryFetchTimeRFC3999(object1.GetName())
		if !ok {
			return binlogLess(folder, object1, object2)
		}
		time2, ok := utility.TryFetchTimeRFC3999(object2.GetName())
		if !ok {
			return binlogLess(folder, object1, object2)
		}
		return time1 < time2
	}
}

func binlogLess(folder storage.Folder, object1, object2 storage.Object) bool {
	binlogName1, ok := tryFetchBinlogName(folder, object1)
	if !ok {
		return false
	}
	binlogName2, ok := tryFetchBinlogName(folder, object2)
	if !ok {
		return false
	}
	return binlogName1 < binlogName2
}

func tryFetchBinlogName(folder storage.Folder, object storage.Object) (string, bool) {
	name := object.GetName()
	if strings.HasPrefix(name, mysql.BinlogPath) {
		_, name = path.Split(name)
		return name, true
	}
	name = strings.Replace(name, utility.SentinelSuffix, "", 1)
	baseBackupFolder := folder.GetSubFolder(utility.BaseBackupPath)
	backup := internal.NewBackup(baseBackupFolder, name)
	var sentinel mysql.StreamSentinelDto
	err := internal.FetchStreamSentinel(backup, &sentinel)
	if err != nil {
		tracelog.InfoLogger.Println("Fail to fetch stream sentinel " + name)
		return "", false
	}
	return sentinel.BinLogStart, true
}
