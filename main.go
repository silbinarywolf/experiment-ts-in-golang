package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

	//"path"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/karrick/godirwalk"
	"github.com/spacemonkeygo/monotime"
)

const typescriptCompilerDir = "typescriptServices/v3.4.5"

type File struct {
	Path    string
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

	// Load typing libs
	// Takes approx ~2ms on Windows
	var libs []File
	{
		err := godirwalk.Walk(typescriptCompilerDir, &godirwalk.Options{
			Callback: func(osPathname string, de *godirwalk.Dirent) error {
				ext := filepath.Ext(osPathname)
				// TODO(Jake): 2019-05-20
				// Need to make this more robust and only check for ".d.ts"
				if ext == ".ts" {
					// NOTE(Jake): 2019-05-20
					// Normalize path in Windows
					osPathname = strings.ReplaceAll(osPathname, "\\", "/")
					content, err := ioutil.ReadFile(osPathname)
					if err != nil {
						log.Fatalln(err)
					}
					libs = append(libs, File{
						Path:    osPathname,
						Content: string(content),
					})
				}
				return nil
			},
			Unsorted: true, // (optional) set true for faster yet non-deterministic enumeration (see godoc)
		})
		if err != nil {
			log.Fatalln(err)
		}
	}

	// Load typescript files
	var files []File
	var filePaths []string
	{
		err := godirwalk.Walk(projectDir, &godirwalk.Options{
			Callback: func(osPathname string, de *godirwalk.Dirent) error {
				ext := filepath.Ext(osPathname)
				if ext == ".ts" ||
					ext == ".tsx" {
					// NOTE(Jake): 2019-05-20
					// Normalize path in Windows
					osPathname = strings.ReplaceAll(osPathname, "\\", "/")
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
				Path:    filePath,
				Content: string(content),
			})
		}
	}

	// Setup TypeScript
	// Takes ~500ms as of 2019-05-20
	var vm *goja.Runtime
	{
		buf, err := ioutil.ReadFile(typescriptCompilerDir + "/typescriptServices.js")
		if err != nil {
			log.Fatalln(err)
		}
		// NOTE: about 4ms
		//fmt.Printf("File load Time taken: %s", time.Since(now))
		vm = goja.New()
		// note(jae): 2021-07-12
		// we do this otherwise the build fails due to missing *.map file for typescriptServices.js
		vm.SetParserOptions(parser.WithDisableSourceMaps)
		if _, err := vm.RunString(string(buf)); err != nil {
			log.Fatalln(err)
		}
	}
	vm.Set("filePath", files[0].Path)
	vm.Set("content", files[0].Content)
	vm.Set("tsconfig", string(tsconfig))

	{
		// NOTE(Jake): 2019-05-20
		// From docs: map[string]interface{} is converted into a host object that largely behaves like a JavaScript Object.
		fileNameToContent := make(map[string]string)
		for _, file := range files {
			fileNameToContent[file.Path] = file.Content
		}
		for _, lib := range libs {
			fileNameToContent[lib.Path] = lib.Content
		}
		vm.Set("FEL_fileNameToContent", fileNameToContent)
		//panic(fmt.Sprintf("%v", fileNameToContent))
		//vm.Set("FEL_fileNameToContent", fileNameToContent)
		vm.Set("FEL_projectDir", projectDir)
		vm.Set("FEL_filePathList", filePaths)
		v, err := vm.RunString(`
var writeFiles = {};

function createCompilerHost(
  options,
  moduleSearchLocations
) {
  function fileExists(fileName) {
    return FEL_fileNameToContent[fileName] !== undefined;
  }

  function readFile(fileName) {
    return FEL_fileNameToContent[fileName];
  }

  function getSourceFile(
    fileName,
    languageVersion,
    onError
  ) {
  	var sourceText = FEL_fileNameToContent[fileName];
  	if (sourceText === undefined) {
  		// todo: Delete this debug line when ready
  		throw new Error('Missing sourceText for: ' + fileName);
  		return undefined;
  	}
  	return ts.createSourceFile(fileName, sourceText, languageVersion);
  }

  function resolveModuleNames(
    moduleNames,
    containingFile
  ){
  	// todo
  	return [];
  	//return [{ resolvedFileName: '' }];
  }

  return {
  	getSourceFile: getSourceFile,
    getDefaultLibFileName: function(options) { return "typescriptServices/v3.4.5/lib.d.ts"; },
    writeFile: function(fileName, content) { writeFiles[fileName] = content; },
    getCurrentDirectory: function() { return FEL_projectDir; },
    getDirectories: function(path) {
    	return [FEL_projectDir]; 
    },
    getCanonicalFileName: function(fileName) { return fileName; },
    getNewLine: function() { return "\n"; },
    useCaseSensitiveFileNames: function() { return true; },
    fileExists: fileExists,
    readFile: readFile,
    resolveModuleNames: resolveModuleNames
  };
}

	var host = createCompilerHost(tsconfig, []);
	var program = ts.createProgram(FEL_filePathList, tsconfig, host);
	var emitResult = program.emit();

	var output = '';
	var allDiagnostics = ts.getPreEmitDiagnostics(program).concat(emitResult.diagnostics);
	allDiagnostics.forEach(function(diagnostic) {
		var message = ts.flattenDiagnosticMessageText(diagnostic.messageText,"\n");
		if (!diagnostic.file) {
			output += message + "\n";
			return;
		}
		output += diagnostic.file.fileName + ":" + message + "\n";
	});
	output;
	`)
		if err != nil {
			log.Fatalln("compile:", err)
		}
		fmt.Printf("%s\nTotal time taken: %s\n", v, time.Since(now))
		panic("end")
	}
	/*{
		v, err := vm.RunString(fmt.Sprintf(`
	var res1 = ts.transpileModule(content, tsconfig);
	res1.outputText;
	`))
		if err != nil {
			log.Fatalln("compile:", err)
		}

		fmt.Printf("%s\nTotal time taken: %s", v, time.Since(now))
	}*/
}
