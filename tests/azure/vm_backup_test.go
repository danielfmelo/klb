package azure_test

import (
	"strings"
	"testing"
	"time"

	"github.com/NeowayLabs/klb/tests/lib/azure"
	"github.com/NeowayLabs/klb/tests/lib/azure/fixture"
)

func TestVMBackup(t *testing.T) {
	t.Parallel()
	vmtesttimeout := 45 * time.Minute
	fixture.Run(t, "VMBackupOsDiskOnly", vmtesttimeout, location, testVMBackupOsDiskOnly)
	fixture.Run(t, "VMBackupStandardLRS", vmtesttimeout, location, testVMBackupStandardLRS)
	fixture.Run(t, "VMBackupPremiumLRS", vmtesttimeout, location, testVMBackupPremiumLRS)
	fixture.Run(t, "VMBackupVMPremiumBackuptStandard", vmtesttimeout, location, testVMBackupVMPremiumBackupStandard)
}

func testVMBackupOsDiskOnly(t *testing.T, f fixture.F) {
	vmSize := "Basic_A2"
	sku := "Standard_LRS"
	backupPrefix := "klb-tests-osdisk"

	resources := createVMResources(t, f)
	vm := createVM(t, f, resources.availSet, resources.nic, vmSize, sku)

	vmBackup := backupVM(t, f, vm, backupPrefix, sku)
	assertResourceGroupExists(t, f, vmBackup)
	defer deleteBackup(t, f, vmBackup)

	recoveredVMName := vm + "2"
	recoverVM(
		t,
		f,
		recoveredVMName,
		resources.vnet,
		resources.subnet,
		vmSize,
		sku,
		vmBackup,
	)
	vms := azure.NewVM(f)
	vms.AssertExistsByName(t, recoveredVMName)
	assertRecoveredVMDisks(t, f, vm, recoveredVMName)
}

func testVMBackupPremiumLRS(t *testing.T, f fixture.F) {
	backupPrefix := "klb-tests-premium"
	testVMBackup(t, f, backupPrefix, "Standard_DS4_v2", "Premium_LRS", "Premium_LRS")
}

func testVMBackupVMPremiumBackupStandard(t *testing.T, f fixture.F) {
	backupPrefix := "klb-tests-premium"
	testVMBackup(t, f, backupPrefix, "Standard_DS4_v2", "Premium_LRS", "Standard_LRS")
}

func testVMBackupStandardLRS(t *testing.T, f fixture.F) {
	backupPrefix := "klb-tests-stdsku"
	testVMBackup(t, f, backupPrefix, "Basic_A2", "Standard_LRS", "Standard_LRS")
}

func testVMBackup(
	t *testing.T,
	f fixture.F,
	backupPrefix string,
	vmSize string,
	vmSKU string,
	backupSKU string,
) {

	resources := createVMResources(t, f)
	vm := createVM(t, f, resources.availSet, resources.nic, vmSize, vmSKU)

	disks := []VMDisk{
		// Different sizes is important to validate behavior
		{Name: genUniqName(), Size: 50, Sku: vmSKU, Caching: "None"},
		{Name: genUniqName(), Size: 100, Sku: vmSKU, Caching: "None"},
	}
	attachDisks(t, f, vm, disks)

	vmBackup := backupVM(t, f, vm, backupPrefix, backupSKU)
	assertResourceGroupExists(t, f, vmBackup)
	defer deleteBackup(t, f, vmBackup)

	backups := listBackups(t, f, vm, backupPrefix)
	assertEqualStringSlice(t, []string{vmBackup}, backups)

	recoveredVMName := vm + "2"
	recoverVM(
		t,
		f,
		recoveredVMName,
		resources.vnet,
		resources.subnet,
		vmSize,
		vmSKU,
		vmBackup,
	)

	assertRecoveredVMDisks(t, f, vm, recoveredVMName)

	recoveredVMBackup := backupVM(t, f, recoveredVMName, backupPrefix, backupSKU)
	assertResourceGroupExists(t, f, recoveredVMBackup)
	defer deleteBackup(t, f, recoveredVMBackup)

	recoveredVMBackups := listBackups(t, f, recoveredVMName, backupPrefix)
	assertEqualStringSlice(t, []string{recoveredVMBackup}, recoveredVMBackups)

	allbackups := listAllBackups(t, f, backupPrefix)
	assertEqualStringSlice(t, []string{vmBackup, recoveredVMBackup}, allbackups)
}

func assertEqualStringSlice(t *testing.T, slice1 []string, slice2 []string) {
	if len(slice2) != len(slice2) {
		t.Fatalf("%+v != %+v", slice1, slice2)
	}

	for _, str1 := range slice1 {
		found := false
		for _, str2 := range slice2 {
			if str1 == str2 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%+v != %+v", slice1, slice2)
		}
	}
}

func assertResourceGroupExists(t *testing.T, f fixture.F, resgroup string) {
	fixture.NewResourceGroup(f.Ctx, t, f.Session, f.Logger).AssertExists(t, resgroup)
}

func assertRecoveredVMDisks(t *testing.T, f fixture.F, vmName string, recoveredVMName string) {
	vm := azure.NewVM(f)
	originalOSDisk := vm.OsDisk(t, vmName)
	restoredOSDisk := vm.OsDisk(t, recoveredVMName)

	// WHY: Names cant be equal
	if originalOSDisk.SizeGB != restoredOSDisk.SizeGB ||
		originalOSDisk.OsType != restoredOSDisk.OsType {
		t.Fatalf("os disk: %+v != %+v", originalOSDisk, restoredOSDisk)
	}

	originalDataDisks := vm.DataDisks(t, vmName)
	recoveredDataDisks := vm.DataDisks(t, recoveredVMName)

	fail := func() {
		t.Fatalf("expected data disks:\n%+v\n\ngot:\n%+v\n\n", originalDataDisks, recoveredDataDisks)
	}

	if len(originalDataDisks) != len(recoveredDataDisks) {
		fail()
	}

	for _, dataDisk := range originalDataDisks {
		found := false
		for _, recoveredDataDisk := range recoveredDataDisks {
			// WHY: Disks cant have the same name
			// LUN is fundamental for backup process, must be identical
			if recoveredDataDisk.Lun == dataDisk.Lun {
				if recoveredDataDisk.SizeGB != dataDisk.SizeGB {
					t.Fatalf(
						"expected disks with same LUN to have same size, %+v != %+v",
						dataDisk,
						recoveredDataDisk,
					)
				}
				found = true
			}
		}

		if !found {
			fail()
		}
	}
}

func backupVM(
	t *testing.T,
	f fixture.F,
	vmname string,
	namespace string,
	sku string,
) string {

	res := execWithIPC(t, f, func(output string) {
		f.Shell.Run(
			"./testdata/backup_vm.sh",
			vmname,
			f.ResGroupName,
			namespace,
			f.Location,
			sku,
			output,
		)
	})
	res = strings.TrimSpace(res)
	return strings.Trim(res, "\n")
}

func parseBackupsList(rawlist string) []string {
	return strings.Split(strings.Trim(strings.TrimSpace(rawlist), "\n"), "\n")
}

func listAllBackups(t *testing.T, f fixture.F, prefix string) []string {

	res := execWithIPC(t, f, func(outputfile string) {
		f.Shell.Run(
			"./testdata/list_all_backups.sh",
			prefix,
			outputfile,
		)
	})

	return parseBackupsList(res)
}

func listBackups(t *testing.T, f fixture.F, vmname string, prefix string) []string {

	res := execWithIPC(t, f, func(outputfile string) {
		f.Shell.Run(
			"./testdata/list_backups.sh",
			vmname,
			prefix,
			outputfile,
		)
	})

	return parseBackupsList(res)
}

func deleteBackup(t *testing.T, f fixture.F, backup string) {
	f.Shell.Run("./testdata/delete_backup.sh", backup)
}

func recoverVM(
	t *testing.T,
	f fixture.F,
	vmName string,
	vnet string,
	subnet string,
	vmSize string,
	sku string,
	backupResgroup string,
) {
	keyFile := "./testdata/key.pub"
	ostype := "linux"

	f.Shell.Run(
		"./testdata/recover_backup.sh",
		vmName,
		f.ResGroupName,
		f.Location,
		vmSize,
		vnet,
		subnet,
		keyFile,
		ostype,
		sku,
		backupResgroup,
	)
}
