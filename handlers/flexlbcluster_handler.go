// Copyright (c) 2022 Yaohui Wang (yaohuiwang@outlook.com)
// FlexLB is licensed under Mulan PubL v2.
// You can use this software according to the terms and conditions of the Mulan PubL v2.
// You may obtain a copy of Mulan PubL v2 at:
//         http://license.coscl.org.cn/MulanPubL-2.0
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
// EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
// MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PubL v2 for more details.

package handlers

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	flexlb "gitee.com/flexlb/flexlb-client-go/client"
	crdv1 "gitee.com/flexlb/flexlb-kube-controller/api/v1"
)

func (h *Handler) ClusterChanged(k8s client.Client, ctx context.Context, cluster *crdv1.FlexLBCluster) error {
	h.lock("update cluster", "cluster", cluster.Name, "handler", "ClusterChanged")
	defer h.unlock("update cluster end", "cluster", cluster.Name, "handler", "ClusterChanged")

	if _, err := h.connectCluster(k8s, ctx, cluster); err != nil {
		return h.errorf(cluster, ErrorClusterNotReady, err, "cluster not ready")
	}

	return nil
}

// connect cluster and refresh status
func (h *Handler) connectCluster(k8s client.Client, ctx context.Context, cluster *crdv1.FlexLBCluster) (*flexlb.Flexlb, error) {
	lb, err1 := flexlb.NewTLSClient(cluster.Spec.Endpoint, h.tlsCaCert, h.tlsClientCert, h.tlsClientKey, h.tlsInsecure, nil)
	if err1 != nil {
		cluster.Status = crdv1.FlexLBClusterStatus{ClusterStatus: crdv1.ClusterStatusNotReady}
		k8s.Status().Update(ctx, cluster)
		return nil, fmt.Errorf("cluster '%s/%s' connect failed: %s", cluster.Namespace, cluster.Name, err1.Error())
	}

	nodeStatus, err2 := lb.GetReadyStatus()
	if err2 != nil {
		cluster.Status = crdv1.FlexLBClusterStatus{ClusterStatus: crdv1.ClusterStatusNotReady}
		k8s.Status().Update(ctx, cluster)
		return nil, fmt.Errorf("cluster '%s/%s' get node ready status failed: %s", cluster.Namespace, cluster.Name, err2.Error())
	}

	cluster.Status = crdv1.FlexLBClusterStatus{ClusterStatus: crdv1.ClusterStatusReady, NodeStatus: nodeStatus}
	return lb, k8s.Status().Update(ctx, cluster)
}
