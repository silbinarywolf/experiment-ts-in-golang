package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"sort"
	"path/filepath"

	"github.com/spacemonkeygo/monotime"
	"github.com/karrick/godirwalk"
	"github.com/dop251/goja"
)

type File struct {
	Path string
	Content string
}

func main() {
	const projectDir = "testdata/.fel"
	now := monotime.Now()

	// Load tsconfig
	tsconfig, err := ioutil.ReadFile(projectDir + "/tsconfig.json")
	if err != nil {
		log.Fatalln(err)
	}

	// Load typescript files
	var files []File
	{
		var filePaths []string
		err := godirwalk.Walk(projectDir, &godirwalk.Options{
	        Callback: func(osPathname string, de *godirwalk.Dirent) error {
	        	ext := filepath.Ext(osPathname)
	        	if ext == ".ts" ||
	        		ext == ".tsx" {
	        		filePaths = append(filePaths, osPathname)
        		}
	            return nil
	        },
	        Unsorted: true, // (optional) set true for faster yet non-deterministic enumeration (see godoc)
	    })
	    if err != nil {
	    	log.Fatalln(err)
	    }
	    sort.Strings(filePaths)
	    for _, filePath := range filePaths {
	    	content, err := ioutil.ReadFile(filePath)
	    	if err != nil {
	    		log.Fatalln(err)
	    	}
	    	files = append(files, File{
	    		Path: filePath,
	    		Content: string(content),
	    	})
	    }
	}

	// Setup TypeScript
	// Takes ~500ms as of 2019-05-20
	var vm *goja.Runtime
	{
		buf, err := ioutil.ReadFile("typescriptServices/v3.4.5/typescriptServices.js")
		if err != nil {
			log.Fatalln(err)
		}
		// NOTE: about 4ms
		//fmt.Printf("File load Time taken: %s", time.Since(now))
		vm = goja.New()
		if _, err := vm.RunString(string(buf)); err != nil {
			log.Fatalln(err)
		}
	}

	vm.Set("content", `
interface Person {
	firstName: string;
	lastName: string;
}

function greeter(person: Person) {
    return "Hello, " + person.firstName + " " + person.lastName;
}
var user = { firstName: "Jane", lastName: "User" };
document.body.innerHTML = greeter(user);`)

	vm.Set("tsconfig", string(tsconfig))

	v, err := vm.RunString(fmt.Sprintf(`
//var compilerOptions = { module: ts.ModuleKind.System };
var res1 = ts.transpileModule(content, tsconfig);
var res2 = ts.transpile(content, tsconfig.compilerOptions, /*fileName*/ undefined, /*diagnostics*/ undefined, /*moduleName*/ "myModule1");
res2;
`))
	if err != nil {
		log.Fatalln("compile:", err)
	}

	fmt.Printf("%s\nTotal time taken: %s", v, time.Since(now))
}
