cmd_/home/amr/Uni/Security_Project/cgo_test/src/module/Module.symvers :=  sed 's/ko$$/o/'  /home/amr/Uni/Security_Project/cgo_test/src/module/modules.order | scripts/mod/modpost -m      -o /home/amr/Uni/Security_Project/cgo_test/src/module/Module.symvers -e -i Module.symvers -T - 
