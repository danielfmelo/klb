package azure

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/NeowayLabs/klb/tests/lib/azure/fixture"
)

type VM struct {
	client compute.VirtualMachinesClient
	f      fixture.F
}

func NewVM(f fixture.F) *VM {
	as := &VM{
		client: compute.NewVirtualMachinesClient(f.Session.SubscriptionID),
		f:      f,
	}
	as.client.Authorizer = f.Session.Token
	return as
}

// AssertAttachedDisk checks if VM has the following disk attached
func (vm *VM) AssertAttachedDataDisk(
	t *testing.T,
	vmname string,
	diskname string,
	diskSizeGB int,
	storageAccountType string,
) {
	vm.f.Retrier.Run(newID("VM", "AssertExists", vmname), func() error {
		v, err := vm.client.Get(vm.f.ResGroupName, vmname, "")
		if err != nil {
			return err
		}
		if v.VirtualMachineProperties == nil {
			return fmt.Errorf("no virtual machine properties found on vm %s", vmname)
		}
		if v.VirtualMachineProperties.StorageProfile == nil {
			return fmt.Errorf("no storage profile found on vm %s", vmname)
		}

		storageProfile := v.VirtualMachineProperties.StorageProfile
		if storageProfile.DataDisks == nil {
			return fmt.Errorf("no data disks found on vm %s", vmname)
		}

		for _, disk := range *storageProfile.DataDisks {
			if disk.Name == nil {
				continue
			}
			if disk.DiskSizeGB == nil {
				continue
			}
			if disk.ManagedDisk == nil {
				continue
			}
			gotName := *disk.Name
			gotDiskSize := int(*disk.DiskSizeGB)
			gotStorageAccountType := string(disk.ManagedDisk.StorageAccountType)

			s.f.Logger.Printf("got disk %q size[%d] %q", gotName, gotDiskSize, gotStorageAccountType)

			if gotName != diskname {
				continue
			}
			if gotDiskSize != diskSizeGB {
				continue
			}
			if gotStorageAccountType != storageAccountType {
				continue
			}

			return nil
		}

		return fmt.Errorf("unable to find disk %q on vm %q", diskname, vmname)
	})
}

// AssertExists checks if VM exists in the resource group.
// Fail tests otherwise.
func (vm *VM) AssertExists(
	t *testing.T,
	name string,
	expectedAvailSet string,
	expectedVMSize string,
	expectedNic string,
) {
	vm.f.Retrier.Run(newID("VM", "AssertExists", name), func() error {
		v, err := vm.client.Get(vm.f.ResGroupName, name, "")
		if err != nil {
			return err
		}
		if v.VirtualMachineProperties == nil {
			return errors.New("Field VirtualMachineProperties is nil!")
		}
		properties := *v.VirtualMachineProperties
		if properties.AvailabilitySet == nil {
			return errors.New("Field AvailabilitySet is nil!")
		}
		if properties.AvailabilitySet.ID == nil {
			return errors.New("Field ID is nil!")
		}
		gotAvailSet := *properties.AvailabilitySet.ID
		if !strings.Contains(gotAvailSet, strings.ToUpper(expectedAvailSet)) {
			return errors.New("AvailSet expected is " + expectedAvailSet + " but got " + gotAvailSet)
		}
		if properties.HardwareProfile == nil {
			return errors.New("Field HardwareProfile is nil!")
		}
		hardwareProfile := *properties.HardwareProfile
		gotVMSize := string(hardwareProfile.VMSize)
		if gotVMSize != expectedVMSize {
			return errors.New("VM Size expected is " + expectedVMSize + " but got " + gotVMSize)
		}
		if properties.StorageProfile == nil {
			return errors.New("Field StorageProfile is nil!")
		}
		if properties.StorageProfile.OsDisk == nil {
			return errors.New("Field OsDisk is nil!")
		}
		if properties.NetworkProfile == nil {
			return errors.New("Field NetworkProfile is nil!")
		}
		network := *properties.NetworkProfile.NetworkInterfaces
		if len(network) == 0 {
			return errors.New("Field NetworkInterfaces is nil!")
		}
		net := network[0]
		if net.ID == nil {
			return errors.New("Field ID is nil!")
		}
		gotNic := string(*net.ID)
		if !strings.Contains(gotNic, expectedNic) {
			return errors.New("Nic expected is " + expectedNic + " but got " + gotNic)
		}
		return nil
	})
}
