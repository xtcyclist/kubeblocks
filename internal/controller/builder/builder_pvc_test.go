/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package builder

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("pvc builder", func() {
	It("should work well", func() {
		const (
			name = "foo"
			ns   = "default"
		)
		resources := corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				"CPU": resource.MustParse("500m"),
			},
		}
		pvc := NewPVCBuilder(ns, name).
			SetResources(resources).
			GetObject()

		Expect(pvc.Name).Should(Equal(name))
		Expect(pvc.Namespace).Should(Equal(ns))
		Expect(pvc.Spec.Resources).Should(Equal(resources))
	})
})
