// Copyright 2022 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef ECAPTURE_H
#define ECAPTURE_H

#ifndef NOCORE
//CO:RE is enabled
#include "vmlinux.h"
#include "bpf/bpf_core_read.h"
#include "bpf/bpf_helpers.h"
#include "bpf/bpf_tracing.h"
#include "bpf/bpf_endian.h"

#else
//CO:RE is disabled
#include <linux/kconfig.h>

// see https://github.com/gojue/ecapture/issues/256 for more detail.
/*
* This will bring in asm_volatile_goto and asm_inline macro definitions
* if enabled by compiler and config options.
*/
#include <linux/types.h>

/*
 * asm_inline is defined as asm __inline in "include/linux/compiler_types.h"
 * if supported by the kernel's CC (i.e CONFIG_CC_HAS_ASM_INLINE) which is not
 * supported by CLANG.
*/
#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif

#include <uapi/linux/ptrace.h>
#include <linux/bpf.h>
#include <linux/socket.h>
#include <net/sock.h>
#include <bpf/bpf_core_read.h>

/*
 * The code in the bpf directory is the same as that in the bpf directory of the Linux kernel source code.
 * move from bpf/bpf_helpers.h to ecapture.h
 * see https://github.com/gojue/ecapture/commit/f50b9de628c9d1c9987d83c1737a673b7a5135b5 more detail.
*/
#if defined(noinline)
#undef noinline
#endif
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_endian.h>
#include <uapi/linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>

// fix  4.19.91-27.7.al7.x86_64/source/include/linux/kernel.h:140:9: warning: 'roundup' macro redefined
#ifndef roundup
#define roundup(x, y) (					\
{							\
	typeof(y) __y = y;				\
	(((x) + (__y - 1)) / __y) * __y;		\
}							\
)
#endif

struct tcphdr {
    __be16 source;
    __be16 dest;
};

#endif

#include "common.h"

#endif
