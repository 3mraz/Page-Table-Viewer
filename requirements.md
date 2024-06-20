# Requirements

This document summarizes the requirements of the project.

## Minimum requirements

Build a Web Application that visualizes the page table of an arbitrary process.
The web server should run in a static executable that works on AMD64.
The view should have a mode where the different paging levels are displayed in a tree view.
Further, add a way to translate a virtual address to a physical one and a way to highlight this mapping in the tree view.
Additionally, live patching of PTEs should be possible in the web app (e.g., flipping the user/kernel bit).

## Better grade

Provide a way to dump and edit the contents of pages.
Further, provide a way to dump the content of all pages that are mapped into the process in a meaningful format (e.g., using json).
Add kernel symbol information (/proc/kallsyms) to the visualization of pages and the page table.

## Even better grade

Detect the data type of page content and pretty-print depending on the page content.
For example, disassemble ELF text sections with the local objdump.

## Best grade

Additionally to supporting AMD64 page tables, support other architectures such as AArch64 and RV64.

## Roadmap

- Play around with PTEditor
- Add go bindings for PTEditor. If harder than thought, use cpp in the following
- Print page table to stdout from go
- Choose a go web framework and display the page table with the previous output
- Use actual html components (such as table) to visualize the page table
- ...
