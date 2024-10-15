# Introduction

This program is designed to inspect and live patch the memory and page tables of a process.
It relies on [PTEditor](https://github.com/misc0110/PTEditor) and it implements a UI for some of its functionalities as well as some extra features such as interactive GDB.

# Project Structure

## src

Contains the code for PTEditor and extra wrapper functions in `utils.c`

## static

Tailwindcss and HTMX.

## templates

All HTML templates.

## utils

- Helper functions for the handlers.
- Written in GO and use the functions of PTEditor.

## handlers

Requests handlers for the webserver.

## main.go

- Handling initialization and cleanup for PTEditor.
- Handlers assignment.

# Used Software

- GO (v1.22.3) for the server and C for PTEditor and some wrapper functions.
- HTMX, Tailwind CSS.
- Node (v18.19.0)
- npm (v9.2.0)

# Running The Project

### Debian

1. Build kernel module. (First time)
   `cd src && make`
2. Load kernel module.
   `sudo insmod src/module/pteditor.ko`
3. Run `./main <port>` (default port 8000).
4. Visit `localhost:<port>`.

For more information about loading the kernel module check [PTEditor](https://github.com/misc0110/PTEditor)

# Development

1. Download GO complier, Node, and npm (preferably the versions used or newer).
2. Download Tailwind CSS or use Tailwind CSS CDN

   #### Download

   `npm install -D tailwindcss`
   Hint: Tailwind CSS build command might not work for older Node versions.

   #### Tailwind CSS CDN (online only)

   1. Comment out the link to the tailwind stylesheet
      `<link rel="stylesheet" href="../static/css/tailwind.css" />`
   2. Uncomment the script tag with the cdn's url
      `<script src="https://cdn.tailwindcss.com"></script>`
   3. Comment out the build-css part in Makefile.

3. Run `make` to build the program.
4. Run `./main <port>` (default port 8000).
5. Visit `localhost:<port>`.
