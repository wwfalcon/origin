package validation

import (
	"net"

	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/util/validation/field"

	oapi "github.com/openshift/origin/pkg/api"
	sdnapi "github.com/openshift/origin/pkg/sdn/api"
)

// ValidateClusterNetwork tests if required fields in the ClusterNetwork are set.
func ValidateClusterNetwork(clusterNet *sdnapi.ClusterNetwork) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&clusterNet.ObjectMeta, false, oapi.MinimalNameRequirements, field.NewPath("metadata"))

	clusterIP, clusterIPNet, err := net.ParseCIDR(clusterNet.Network)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("network"), clusterNet.Network, err.Error()))
	} else {
		ones, bitSize := clusterIPNet.Mask.Size()
		if (bitSize - ones) <= clusterNet.HostSubnetLength {
			allErrs = append(allErrs, field.Invalid(field.NewPath("hostSubnetLength"), clusterNet.HostSubnetLength, "subnet length is greater than cluster Mask"))
		}
	}

	serviceIP, serviceIPNet, err := net.ParseCIDR(clusterNet.ServiceNetwork)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("serviceNetwork"), clusterNet.ServiceNetwork, err.Error()))
	}

	if (clusterIPNet != nil) && (serviceIP != nil) && clusterIPNet.Contains(serviceIP) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("serviceNetwork"), clusterNet.ServiceNetwork, "service network overlaps with cluster network"))
	}
	if (serviceIPNet != nil) && (clusterIP != nil) && serviceIPNet.Contains(clusterIP) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("network"), clusterNet.Network, "cluster network overlaps with service network"))
	}

	return allErrs
}

func validateNewNetwork(obj *sdnapi.ClusterNetwork, old *sdnapi.ClusterNetwork) *field.Error {
	oldBase, oldNet, err := net.ParseCIDR(old.Network)
	if err != nil {
		// Shouldn't happen, but if the existing value is invalid, then any change should be an improvement...
		return nil
	}
	oldSize, _ := oldNet.Mask.Size()
	_, newNet, err := net.ParseCIDR(obj.Network)
	if err != nil {
		return field.Invalid(field.NewPath("network"), obj.Network, err.Error())
	}
	newSize, _ := newNet.Mask.Size()
	// oldSize/newSize is, eg the "16" in "10.1.0.0/16", so "newSize < oldSize" means
	// the new network is larger
	if newSize < oldSize && newNet.Contains(oldBase) {
		return nil
	} else {
		return field.Invalid(field.NewPath("network"), obj.Network, "cannot change the cluster's network CIDR to a value that does not include the existing network.")
	}
}

func ValidateClusterNetworkUpdate(obj *sdnapi.ClusterNetwork, old *sdnapi.ClusterNetwork) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))

	if obj.Network != old.Network {
		err := validateNewNetwork(obj, old)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if obj.HostSubnetLength != old.HostSubnetLength {
		allErrs = append(allErrs, field.Invalid(field.NewPath("hostSubnetLength"), obj.HostSubnetLength, "cannot change the cluster's hostSubnetLength midflight."))
	}
	if obj.ServiceNetwork != old.ServiceNetwork && old.ServiceNetwork != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("serviceNetwork"), obj.ServiceNetwork, "cannot change the cluster's serviceNetwork CIDR midflight."))
	}

	return allErrs
}

// ValidateHostSubnet tests fields for the host subnet, the host should be a network resolvable string,
//  and subnet should be a valid CIDR
func ValidateHostSubnet(hs *sdnapi.HostSubnet) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&hs.ObjectMeta, false, oapi.MinimalNameRequirements, field.NewPath("metadata"))

	_, _, err := net.ParseCIDR(hs.Subnet)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("subnet"), hs.Subnet, err.Error()))
	}
	if net.ParseIP(hs.HostIP) == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("hostIP"), hs.HostIP, "invalid IP address"))
	}
	return allErrs
}

func ValidateHostSubnetUpdate(obj *sdnapi.HostSubnet, old *sdnapi.HostSubnet) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))

	if obj.Subnet != old.Subnet {
		allErrs = append(allErrs, field.Invalid(field.NewPath("subnet"), obj.Subnet, "cannot change the subnet lease midflight."))
	}

	return allErrs
}

// ValidateNetNamespace tests fields for a greater-than-zero NetID
func ValidateNetNamespace(netnamespace *sdnapi.NetNamespace) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&netnamespace.ObjectMeta, false, oapi.MinimalNameRequirements, field.NewPath("metadata"))

	if netnamespace.NetID < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("netID"), netnamespace.NetID, "invalid Net ID: cannot be negative"))
	}
	return allErrs
}

func ValidateNetNamespaceUpdate(obj *sdnapi.NetNamespace, old *sdnapi.NetNamespace) field.ErrorList {
	return validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
}
