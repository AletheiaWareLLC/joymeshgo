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
	"math"
	"os"
	"strconv"
	"strings"
)

var vertices uint32
var vertex []float64
var vertexNormal []float64
var textureCoordinate []float64

var tempFace [][3]int
var tempVertex [][]float64
var tempNormal [][]float64
var tempTextureCoordinate [][]float64

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) > 2 {
		// Reads .obj or .off files, creates Mesh and writes to stdout or output file
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
					tempTextureCoordinate = append(tempTextureCoordinate, stringsToFloats(parts[1:4]...))
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
			scanner.Scan()
			line := scanner.Text()
			if !strings.HasPrefix(line, "OFF") {
				log.Fatal("Invalid OFF file")
			}
			log.Println(line)
			parts := strings.Fields(line)
			vertexCount := -1
			faceCount := -1
			if len(parts) > 2 {
				vertexCount = stringToInt(parts[1])
				faceCount = stringToInt(parts[2])
			}
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
						count := stringToInt(parts[0])
						if count > 3 {
							// Convert Polygon to Triangles
							for i := 1; i < count-1; i++ {
								addOffFace(parts[1], parts[i+1], parts[i+2])
							}
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
			calculateOffNormals(vertexCount)
		} else {
			log.Fatal("Unsupported File:", path)
		}

		mesh := &joygo.Mesh{
			Name:     name,
			Vertices: vertices,
			Vertex:   vertex,
			Normal:   vertexNormal,
			TexCoord: textureCoordinate,
		}
		if len(os.Args) > 3 {
			file, err := os.OpenFile(os.Args[3], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			size := uint64(proto.Size(mesh))
			data, err := proto.Marshal(mesh)
			if err != nil {
				log.Fatal(err)
			}
			if _, err := file.Write(proto.EncodeVarint(size)); err != nil {
				log.Fatal(err)
			}
			if _, err := file.Write(data); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println(proto.MarshalTextString(mesh))
		}
	} else {
		log.Println("Joy Mesh Usage:")
		log.Println("\tjoymesh <mesh-name> <input-file> (write to stdout)")
		log.Println("\tjoymesh <mesh-name> <input-file> <output-file>")
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
			ti := objIndexToArrayIndex(stringToInt(index[1]), len(tempTextureCoordinate))
			textureCoordinate = append(textureCoordinate, tempTextureCoordinate[ti]...)
		}
		if len(index) > 2 && index[2] != "" {
			ni := objIndexToArrayIndex(stringToInt(index[2]), len(tempNormal))
			vertexNormal = append(vertexNormal, tempNormal[ni]...)
		}
	}
}

func addOffFace(ss ...string) {
	var face [3]int
	var vs [3][]float64
	for i := 0; i < 3; i++ {
		face[i] = stringToInt(ss[i])
		vs[i] = tempVertex[face[i]]
		vertex = append(vertex, vs[i]...)
		vertices++
	}
	tempFace = append(tempFace, face)
	// calculate face edges
	e0 := [3]float64{
		vs[1][0] - vs[0][0],
		vs[1][1] - vs[0][1],
		vs[1][2] - vs[0][2],
	}
	e1 := [3]float64{
		vs[2][0] - vs[0][0],
		vs[2][1] - vs[0][1],
		vs[2][2] - vs[0][2],
	}
	// calculate face normal
	n0 := e0[1]*e1[2] - e0[2]*e1[1]
	n1 := e0[2]*e1[0] - e0[0]*e1[2]
	n2 := e0[0]*e1[1] - e0[1]*e1[0]
	// normalize face normal
	length := math.Sqrt((n0 * n0) + (n1 * n1) + (n2 * n2))
	if length > 0 {
		n0 = n0 / length
		n1 = n1 / length
		n2 = n2 / length
	}
	n := []float64{
		n0,
		n1,
		n2,
	}
	tempNormal = append(tempNormal, n)
}

func calculateOffNormals(vertexCount int) {
	if os.Getenv("NORMALS") == "smooth" {
		log.Println("Calculating smooth normals")
		vns := make(map[int][]float64, vertexCount)
		// Loop once to add face normals to vertex normals
		for f, face := range tempFace {
			fn := tempNormal[f]
			for i := 0; i < 3; i++ {
				vn := vns[face[i]]
				if vn == nil {
					vn = []float64{0, 0, 0}
				}
				vn[0] += fn[0]
				vn[1] += fn[1]
				vn[2] += fn[2]
				vns[face[i]] = vn
			}
		}
		// Loop again to add tempVertexNormals to vertexNormal
		for _, face := range tempFace {
			for i := 0; i < 3; i++ {
				vn := vns[face[i]]
				// Normalize normal
				length := math.Sqrt((vn[0] * vn[0]) + (vn[1] * vn[1]) + (vn[2] * vn[2]))
				if length > 0 {
					vn[0] /= length
					vn[1] /= length
					vn[2] /= length
				}
				vertexNormal = append(vertexNormal, vn...)
			}
		}
	} else {
		for _, temp := range tempNormal {
			// Add once for each vertex
			for i := 0; i < 3; i++ {
				vertexNormal = append(vertexNormal, temp...)
			}
		}
	}
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
