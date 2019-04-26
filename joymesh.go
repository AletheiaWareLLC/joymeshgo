/*
 * Copyright 2019 Aletheia Ware LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"github.com/AletheiaWareLLC/joygo"
	"github.com/golang/protobuf/proto"
	"log"
	"os"
	"strconv"
	"strings"
)

var vertices uint32
var vertex []float64
var normal []float64
var texCoord []float64

var tempVertex [][]float64
var tempNormal [][]float64
var tempTexCoord [][]float64

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) > 2 {
		// TODO read .obj or .off file, create Mesh and write to stdout
		// filename.obj or filename.off
		name := os.Args[1]
		path := os.Args[2]
		file, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		vertices = 0

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		if strings.HasSuffix(path, ".obj") {
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				//log.Println(line)
				parts := strings.Split(line, " ")
				switch parts[0] {
				case "v":
					tempVertex = append(tempVertex, stringsToFloats(parts[1:4]...))
				case "vn":
					tempNormal = append(tempNormal, stringsToFloats(parts[1:4]...))
				case "vt":
					tempTexCoord = append(tempTexCoord, stringsToFloats(parts[1:4]...))
				case "f":
					if len(parts) > 4 {
						// Convert Quad to Triangles
						addObjFace(parts[1:4]...)
						addObjFace(parts[3], parts[4], parts[1])
					} else {
						// Handle triangle
						addObjFace(parts[1:4]...)
					}
				case "l":
					addObjLine(parts[1:3]...)
				default:
					log.Println("Ignoring:", line)
				}
			}
		} else if strings.HasSuffix(path, ".off") {
			line := scanner.Text()
			if line != "OFF" {
				log.Fatal("Invalid OFF file")
			}
			vertexCount := -1
			faceCount := -1
			log.Println(line)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.Fields(line)
				if vertexCount < 0 && faceCount < 0 {
					vertexCount = stringToInt(parts[0])
					faceCount = stringToInt(parts[1])
				} else {
					if vertexCount > 0 {
						tempVertex = append(tempVertex, stringsToFloats(parts[0:3]...))
						vertexCount--
					} else if faceCount > 0 {
						if parts[0] == "4" {
							// Convert Quad to Triangles
							addOffFace(parts[1:4]...)
							addOffFace(parts[3], parts[4], parts[1])
						} else {
							// Handle triangle
							addOffFace(parts[1:4]...)
						}
						faceCount--
					} else {
						log.Println("Ignoring:", line)
					}
				}
			}
		} else {
			log.Fatal("Unsupported File:", path)
		}

		mesh := joygo.Mesh{
			Name:     name,
			Vertices: vertices,
			Vertex:   vertex,
			Normal:   normal,
			TexCoord: texCoord,
		}
		log.Println(proto.MarshalTextString(&mesh))
	} else {
		log.Println("Joy Mesh Usage:")
		log.Println("\tjoymesh <mesh-name> <file-path>")
	}
}

func addObjLine(ss ...string) {
	for i := 0; i < 2; i++ {
		index := objIndexToArrayIndex(stringToInt(ss[i]), len(tempVertex))
		vertex = append(vertex, tempVertex[index]...)
		vertices++
	}
}

func addObjFace(ss ...string) {
	for i := 0; i < 3; i++ {
		index := strings.Split(ss[i], "/")
		vi := objIndexToArrayIndex(stringToInt(index[0]), len(tempVertex))
		vertex = append(vertex, tempVertex[vi]...)
		vertices++
		if len(index) > 1 && index[1] != "" {
			ti := objIndexToArrayIndex(stringToInt(index[1]), len(tempTexCoord))
			texCoord = append(texCoord, tempTexCoord[ti]...)
		}
		if len(index) > 2 && index[2] != "" {
			ni := objIndexToArrayIndex(stringToInt(index[2]), len(tempNormal))
			normal = append(normal, tempNormal[ni]...)
		}
	}
}

func addOffFace(ss ...string) {
	var face [3][]float64
	for i := 0; i < 3; i++ {
		index := stringToInt(ss[i])
		face[i] = tempVertex[index]
		vertex = append(vertex, face[i]...)
		vertices++
	}
	n := calculateNormal(face[0], face[1], face[2])
	for i := 0; i < 3; i++ {
		normal = append(normal, n...)
	}
}

func calculateNormal(v1, v2, v3 []float64) []float64 {
	var normal []float64
	// TODO var u [3]float64
	// TODO var v [3]float64
	// TODO
	return normal
}

func stringToInt(s string) int {
	index, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return index
}

func stringToFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func stringsToFloats(ss ...string) []float64 {
	count := len(ss)
	fs := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		fs = append(fs, stringToFloat(ss[i]))
	}
	return fs
}

func objIndexToArrayIndex(index, size int) int {
	if index < 0 {
		// OBJ negative indices start from end of list, -1 == last element
		return size - index
	}
	// OBJ vertex indices start at 1
	return index - 1
}
