package azure_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NeowayLabs/klb/tests/lib/assert"
	"github.com/NeowayLabs/klb/tests/lib/azure"
	"github.com/NeowayLabs/klb/tests/lib/azure/fixture"
)

func TestStorage(t *testing.T) {
	timeout := 10 * time.Minute
	t.Parallel()
	fixture.Run(
		t,
		"StorageAccountCreateStandardLRS",
		timeout,
		location,
		testStorageAccountCreateStandardLRS,
	)
	fixture.Run(
		t,
		"StorageAccountCreatePremiumLRS",
		timeout,
		location,
		testStorageAccountCreatePremiumLRS,
	)
	fixture.Run(
		t,
		"StorageAccountCreateBLOBHot",
		timeout,
		location,
		testStorageAccountCreateBLOBHot,
	)
	fixture.Run(
		t,
		"StorageAccountCreateBLOBCold",
		timeout,
		location,
		testStorageAccountCreateBLOBCold,
	)
	fixture.Run(
		t,
		"StorageAccountUploadFiles",
		timeout,
		location,
		testStorageAccountUploadFiles,
	)
	fixture.Run(
		t,
		"StorageAccountCheckResourcesExistence",
		timeout,
		location,
		testStorageAccountCheckResourcesExistence,
	)
	fixture.Run(
		t,
		"BlobFSDirOperations",
		timeout,
		location,
		testBlobFSUploadsDirOperations,
	)
	fixture.Run(
		t,
		"BlobFSUploadsWhenAccountAndContainerExists",
		timeout,
		location,
		testBlobFSUploadsWhenAccountAndContainerExists,
	)
	testBlobFSCreatesAccountAndContainerIfNonExistent(
		t,
		timeout,
		location,
	)
}

func testStorageAccountCreateBLOBHot(t *testing.T, f fixture.F) {
	sku := "Standard_LRS"
	tier := "Hot"
	kind := "BlobStorage"
	name := genStorageAccountName()

	createStorageAccountBLOB(f, name, sku, tier)
	checkStorageBlobAccount(t, f, name, sku, tier, kind)
}

func testStorageAccountCreateBLOBCold(t *testing.T, f fixture.F) {
	sku := "Standard_LRS"
	tier := "Cool"
	kind := "BlobStorage"
	name := genStorageAccountName()

	createStorageAccountBLOB(f, name, sku, tier)
	checkStorageBlobAccount(t, f, name, sku, tier, kind)
}

func testStorageAccountCreateStandardLRS(t *testing.T, f fixture.F) {
	sku := "Standard_LRS"
	kind := "Storage"
	name := genStorageAccountName()

	createStorageAccount(f, name, sku)
	checkStorageAccount(t, f, name, sku, kind)
}

func testStorageAccountCreatePremiumLRS(t *testing.T, f fixture.F) {
	sku := "Premium_LRS"
	kind := "Storage"
	name := genStorageAccountName()

	createStorageAccount(f, name, sku)
	checkStorageAccount(t, f, name, sku, kind)
}

type BlobStorageFixture struct {
	sku             string
	kind            string
	tier            string
	account         string
	container       string
	testfile        string
	testfileContent string
}

func setupBlobStorageFixture(t *testing.T, f fixture.F, sku string, tier string) (BlobStorageFixture, func()) {
	container := fixture.NewUniqueName("container")

	expectedContent := fixture.NewUniqueName("random-content")
	localfile, cleanup := setupTestFile(t, expectedContent)

	account := genStorageAccountName()
	return BlobStorageFixture{
		sku:             sku,
		kind:            "BlobStorage",
		tier:            tier,
		account:         account,
		container:       container,
		testfile:        localfile,
		testfileContent: expectedContent,
	}, cleanup
}

func testBlobFSUploadsDirOperations(t *testing.T, f fixture.F) {
	account := genStorageAccountName()
	container := fixture.NewUniqueName("container")

	type TestFile struct {
		path    string
		content string
	}

	createFilesOnDir := func(dir string, filesCount int) []TestFile {
		assert.NoError(t, os.MkdirAll(dir, 0644))
		files := []TestFile{}
		for i := 0; i < filesCount; i++ {
			localfile := filepath.Join(dir, strconv.Itoa(i))
			content := fixture.NewUniqueName("somecontent")
			ioutil.WriteFile(localfile, []byte(content), 0644)
			files = append(files, TestFile{
				path:    localfile,
				content: content,
			})
		}
		return files
	}

	tdir, err := ioutil.TempDir("", "uploader-recur-test")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tdir))
	}()

	localfiles := []TestFile{}
	localfiles = append(localfiles, createFilesOnDir(
		filepath.Join(tdir), 2)...)
	localfiles = append(localfiles, createFilesOnDir(
		filepath.Join(tdir, "level1"), 1)...)
	localfiles = append(localfiles, createFilesOnDir(
		filepath.Join(tdir, "level1", "level2"), 3)...)

	sku := "Standard_LRS"
	tier := "Cool"
	kind := "BlobStorage"
	remotedir := "/remote/path"

	fs := newBlobFS(account, sku, tier, container)
	fs.UploadDir(t, f, remotedir, tdir)

	checkStorageBlobAccount(t, f, account, sku, tier, kind)

	expectedRemoteFiles := []string{}
	remoteToLocalFile := map[string]TestFile{}

	for _, file := range localfiles {
		remotepath := filepath.Join(
			remotedir,
			strings.TrimPrefix(file.path, tdir))
		remoteToLocalFile[remotepath] = file
		expectedRemoteFiles = append(expectedRemoteFiles, remotepath)
	}

	testBlobFSListDir(t, f, fs, remotedir, expectedRemoteFiles)

	for remotepath, file := range remoteToLocalFile {
		filecontent := downloadFileBLOB(t, f, account, container, remotepath)
		assert.EqualStrings(
			t,
			file.content,
			filecontent,
			"checking uploaded BLOB individually")
	}
}

func testBlobFSListDir(
	t *testing.T,
	f fixture.F,
	fs blobFS,
	remotedir string,
	expectedFiles []string,
) {
	// TODO
}

func testBlobFSUploadsWhenAccountAndContainerExists(t *testing.T, f fixture.F) {
	sf, cleanup := setupBlobStorageFixture(t, f, "Standard_LRS", "Cool")
	defer cleanup()

	expectedPath := "/test/acc/container/existent/file"
	createStorageAccountBLOB(f, sf.account, sf.sku, sf.tier)
	checkStorageBlobAccount(t, f, sf.account, sf.sku, sf.tier, sf.kind)
	createStorageAccountContainer(f, sf.account, sf.container)

	blobFSUpload(
		t,
		f,
		sf.account,
		"Premium_LRS",
		"Hot",
		sf.container,
		expectedPath,
		sf.testfile,
	)

	checkStorageBlobAccount(t, f, sf.account, sf.sku, sf.tier, sf.kind)
	filecontent := downloadFileBLOB(t, f, sf.account, sf.container, expectedPath)
	assert.EqualStrings(t, sf.testfileContent, filecontent, "checking uploaded BLOB")
}

func uploaderCreatesAccountAndContainerIfNonExistent(
	t *testing.T,
	f fixture.F,
	sku string,
	tier string,
) {
	sf, cleanup := setupBlobStorageFixture(t, f, sku, tier)
	defer cleanup()

	expectedPath := "/test/acc/container/nonexistent/file"

	blobFSUpload(
		t,
		f,
		sf.account,
		sf.sku,
		sf.tier,
		sf.container,
		expectedPath,
		sf.testfile,
	)
	checkStorageBlobAccount(t, f, sf.account, sf.sku, sf.tier, sf.kind)
	filecontent := downloadFileBLOB(t, f, sf.account, sf.container, expectedPath)

	assert.EqualStrings(t, sf.testfileContent, filecontent, "checking uploaded BLOB")
}

func testBlobFSCreatesAccountAndContainerIfNonExistent(
	t *testing.T,
	timeout time.Duration,
	location string,
) {
	type TestCase struct {
		sku  string
		tier string
	}

	// WHY: lot of cases are not tested because they do not work on az cli
	// like using Premium_LRS or Standard_ZRS as SKU.
	// The docs are pretty confusing right now
	// on how to use the SKU option when handling blob storage.
	tcases := []TestCase{
		TestCase{
			sku:  "Standard_LRS",
			tier: "Cool",
		},
		TestCase{
			sku:  "Standard_LRS",
			tier: "Hot",
		},
		TestCase{
			sku:  "Standard_GRS",
			tier: "Cool",
		},
		TestCase{
			sku:  "Standard_GRS",
			tier: "Hot",
		},
		TestCase{
			sku:  "Standard_RAGRS",
			tier: "Cool",
		},
		TestCase{
			sku:  "Standard_RAGRS",
			tier: "Hot",
		},
	}

	for _, tcase := range tcases {
		tname := fmt.Sprintf(
			"UploaderCreatesAccountAndContainerIfNonExistent%s%s",
			tcase.sku,
			tcase.tier,
		)
		fixture.Run(t, tname, timeout, location, func(t *testing.T, f fixture.F) {
			uploaderCreatesAccountAndContainerIfNonExistent(t, f, tcase.sku, tcase.tier)
		})
	}
}

func testStorageAccountUploadFiles(t *testing.T, f fixture.F) {

	sf, cleanup := setupBlobStorageFixture(t, f, "Standard_LRS", "Cool")
	defer cleanup()

	createStorageAccountBLOB(f, sf.account, sf.sku, sf.tier)
	checkStorageBlobAccount(t, f, sf.account, sf.sku, sf.tier, sf.kind)

	createStorageAccountContainer(f, sf.account, sf.container)

	expectedPath := "/account/file"
	uploadFileBLOB(t, f, sf.account, sf.container, expectedPath, sf.testfile)
	contents := downloadFileBLOB(t, f, sf.account, sf.container, expectedPath)

	assert.EqualStrings(t, sf.testfileContent, contents, "downloading BLOB")
}

func testStorageAccountCheckResourcesExistence(t *testing.T, f fixture.F) {
	sku := "Standard_LRS"
	kind := "BlobStorage"
	tier := "Cool"
	accountname := genStorageAccountName()

	//WHY: Testing multiple things on one test is bad but
	//     building the context to run tests is too expensive on the cloud
	testStorageAccountDontExist(t, f, accountname)
	createStorageAccountBLOB(f, accountname, sku, tier)
	checkStorageBlobAccount(t, f, accountname, sku, tier, kind)
	testStorageAccountExists(t, f, accountname)

	containerName := "klb-test-container-exists"
	testContainerDontExist(t, f, accountname, containerName)
	createStorageAccountContainer(f, accountname, containerName)
	testContainerExists(t, f, accountname, containerName)

	remotepath := "/test/exists"
	localfile, cleanup := setupTestFile(t, "whatever")
	defer cleanup()

	testBLOBDontExist(t, f, accountname, containerName, remotepath)
	uploadFileBLOB(t, f, accountname, containerName, remotepath, localfile)
	testBLOBExists(t, f, accountname, containerName, remotepath)
}

type blobFS struct {
	account   string
	sku       string
	tier      string
	container string
}

func newBlobFS(
	account string,
	sku string,
	tier string,
	container string,
) blobFS {
	return blobFS{
		account:   account,
		sku:       sku,
		tier:      tier,
		container: container,
	}
}

func (fs *blobFS) List(t *testing.T, f fixture.F, remotedir string) []string {
	res := execWithIPC(t, f, func(localpath string) {
		f.Shell.Run(
			"./testdata/blob_fs_list.sh",
			f.ResGroupName,
			fs.account,
			fs.container,
			remotedir,
		)
	})
	return strings.Split(res, "\n")
}

func (fs *blobFS) UploadDir(
	t *testing.T,
	f fixture.F,
	remotedir string,
	localdir string,
) {
	f.Shell.Run(
		"./testdata/blob_fs_upload_dir.sh",
		f.ResGroupName,
		f.Location,
		fs.account,
		fs.sku,
		fs.tier,
		fs.container,
		remotedir,
		localdir,
	)
}

func blobFSUpload(
	t *testing.T,
	f fixture.F,
	account string,
	sku string,
	tier string,
	container string,
	remotepath string,
	localpath string,
) {
	f.Shell.Run(
		"./testdata/blob_fs_upload.sh",
		f.ResGroupName,
		f.Location,
		account,
		sku,
		tier,
		container,
		remotepath,
		localpath,
	)
}

func uploadFileBLOB(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
	remotepath string,
	localpath string,
) {
	f.Shell.Run(
		"./testdata/storage_upload_blob.sh",
		f.ResGroupName,
		account,
		container,
		remotepath,
		localpath,
	)
}

func downloadFileBLOB(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
	remotepath string,
) string {
	// FIXME: execWithIPC trims the contents, not a problem for now
	return execWithIPC(t, f, func(localpath string) {
		f.Shell.Run(
			"./testdata/storage_download_blob.sh",
			f.ResGroupName,
			account,
			container,
			remotepath,
			localpath,
		)
	})
}

func setupTestFile(t *testing.T, expectedContents string) (string, func()) {
	f, err := ioutil.TempFile("", "az-storage-account-tests")
	assert.NoError(t, err, "creating tmp file")

	n, err := io.Copy(f, bytes.NewBufferString(expectedContents))
	assert.NoError(t, err, "copying to tmp file")
	assert.EqualInts(t, len(expectedContents), int(n), "copying to tmp file")

	cleanup := func() {
		assert.NoError(t, os.Remove(f.Name()), "removing tmp file")
	}

	return f.Name(), cleanup
}

func genStorageAccountName() string {
	return strings.Replace(fixture.NewUniqueName("st"), "-", "", -1)
}

func createStorageAccount(
	f fixture.F,
	name string,
	sku string,
) {
	f.Shell.Run(
		"./testdata/create_storage_account.sh",
		f.ResGroupName,
		name,
		f.Location,
		sku,
	)
}

func testStorageAccountExists(t *testing.T, f fixture.F, account string) {
	err := storageAccountExists(f, account)
	if err != nil {
		t.Fatalf("expected account[%s] to exist, error[%s]", account, err)
	}
}

func testStorageAccountDontExist(t *testing.T, f fixture.F, account string) {
	err := storageAccountExists(f, account)
	if err == nil {
		t.Fatalf("expected account[%s] to not exist", account)
	}
}

func testContainerExists(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
) {
	err := containerExists(f, account, container)
	if err != nil {
		t.Fatalf("expected container[%s] to exist, error[%s]", container, err)
	}
}

func testContainerDontExist(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
) {
	err := containerExists(f, account, container)
	if err == nil {
		t.Fatalf("expected container[%s] to not exist", container)
	}
}

func containerExists(
	f fixture.F,
	account string,
	container string,
) error {
	return f.Shell.RunOnce(
		"./testdata/storage_container_exists.sh",
		f.ResGroupName,
		account,
		container,
	)
}

func testBLOBExists(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
	remotepath string,
) {
	err := blobExists(f, account, container, remotepath)
	if err != nil {
		t.Fatalf("expected path[%s] to exist, error[%s]", remotepath, err)
	}
}

func testBLOBDontExist(
	t *testing.T,
	f fixture.F,
	account string,
	container string,
	remotepath string,
) {
	err := blobExists(f, account, container, remotepath)
	if err == nil {
		t.Fatalf("expected path[%s] to not exist", remotepath)
	}
}

func blobExists(
	f fixture.F,
	account string,
	container string,
	remotepath string,
) error {
	return f.Shell.RunOnce(
		"./testdata/storage_blob_exists.sh",
		f.ResGroupName,
		account,
		container,
		remotepath,
	)
}

func storageAccountExists(
	f fixture.F,
	account string,
) error {
	return f.Shell.RunOnce(
		"./testdata/storage_account_exists.sh",
		f.ResGroupName,
		account,
	)
}

func createStorageAccountBLOB(
	f fixture.F,
	name string,
	sku string,
	accessTier string,
) {
	f.Shell.Run(
		"./testdata/create_storage_account_blob.sh",
		f.ResGroupName,
		name,
		f.Location,
		sku,
		accessTier,
	)
}

func createStorageAccountContainer(f fixture.F, account string, container string) {
	f.Shell.Run(
		"./testdata/create_storage_container.sh",
		f.ResGroupName,
		account,
		container,
	)
}

func checkStorageAccount(
	t *testing.T,
	f fixture.F,
	name string,
	sku string,
	kind string,
) {
	accounts := azure.NewStorageAccounts(f)
	account := accounts.Account(t, name)

	assert.EqualStrings(t, name, account.Name, "checking name")
	assert.EqualStrings(t, f.Location, account.Location, "checking location")
	assert.EqualStrings(t, sku, account.Sku, "checking SKU")
	assert.EqualStrings(t, kind, account.Kind, "checking kind")
}

func checkStorageBlobAccount(
	t *testing.T,
	f fixture.F,
	name string,
	sku string,
	tier string,
	kind string,
) {
	accounts := azure.NewStorageAccounts(f)
	account := accounts.Account(t, name)

	assert.EqualStrings(t, name, account.Name, "checking name")
	assert.EqualStrings(t, f.Location, account.Location, "checking location")
	assert.EqualStrings(t, kind, account.Kind, "checking kind")
	assert.EqualStrings(t, sku, account.Sku, "checking kind")
	assert.EqualStrings(t, tier, account.AccessTier, "checking tier")
}
