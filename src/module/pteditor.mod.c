#include <linux/module.h>
#define INCLUDE_VERMAGIC
#include <linux/build-salt.h>
#include <linux/elfnote-lto.h>
#include <linux/export-internal.h>
#include <linux/vermagic.h>
#include <linux/compiler.h>

BUILD_SALT;
BUILD_LTO_INFO;

MODULE_INFO(vermagic, VERMAGIC_STRING);
MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};

#ifdef CONFIG_RETPOLINE
MODULE_INFO(retpoline, "Y");
#endif


static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0x7c339fa3, "pv_ops" },
	{ 0x5b8239ca, "__x86_return_thunk" },
	{ 0xbdfb6dbb, "__fentry__" },
	{ 0x78d27962, "boot_cpu_data" },
	{ 0x65487097, "__x86_indirect_thunk_rax" },
	{ 0xd0da656b, "__stack_chk_fail" },
	{ 0xfcca5424, "register_kprobe" },
	{ 0x63026490, "unregister_kprobe" },
	{ 0x92997ed8, "_printk" },
	{ 0x1aa4f739, "misc_register" },
	{ 0xacfde3ae, "register_kretprobe" },
	{ 0x30a0c8ea, "proc_create" },
	{ 0xe43953af, "current_task" },
	{ 0xed00154b, "find_vpid" },
	{ 0x44a68657, "pid_task" },
	{ 0x5a5a2271, "__cpu_online_mask" },
	{ 0x63f835ba, "on_each_cpu_cond_mask" },
	{ 0x2bcc6bf1, "misc_deregister" },
	{ 0x50adaa18, "unregister_kretprobe" },
	{ 0x9526406a, "remove_proc_entry" },
	{ 0x72d79d83, "pgdir_shift" },
	{ 0x1d19f77b, "physical_mask" },
	{ 0x8a35b432, "sme_me_mask" },
	{ 0xdad13544, "ptrs_per_p4d" },
	{ 0x7cd8d75e, "page_offset_base" },
	{ 0xb9fa2d5f, "__tracepoint_mmap_lock_start_locking" },
	{ 0x668b19a1, "down_read" },
	{ 0x1cf55010, "__tracepoint_mmap_lock_acquire_returned" },
	{ 0xfbca24c4, "__tracepoint_mmap_lock_released" },
	{ 0x53b954a2, "up_read" },
	{ 0x21ad6194, "__mmap_lock_do_trace_acquire_returned" },
	{ 0x1f0050da, "__mmap_lock_do_trace_released" },
	{ 0x74a1f1e9, "__mmap_lock_do_trace_start_locking" },
	{ 0xecdcabd2, "copy_user_generic_unrolled" },
	{ 0x1f199d24, "copy_user_generic_string" },
	{ 0x21271fd0, "copy_user_enhanced_fast_string" },
	{ 0x57bc19d2, "down_write" },
	{ 0xce807a25, "up_write" },
	{ 0x4c9d28b0, "phys_base" },
	{ 0xc4ae50da, "module_layout" },
};

MODULE_INFO(depends, "");

