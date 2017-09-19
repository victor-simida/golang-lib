package nameUtil

import (
	"log"
	"strings"
)

/*根据两个字符串的相似度，判断字符串是否模糊匹配，Levenshtein Distance算法*/
func NameMatch(match, origin string, standard float64) bool {
	m := len(match)
	n := len(origin)

	if m == 0 || n == 0 {
		return false
	}

	match = strings.ToLower(match)
	origin = strings.ToLower(origin)

	matrix := newMatrix(m, n)

	for i := 1; i <= m; i++ {
		ch1 := match[i-1]
		for j := 1; j <= n; j++ {
			ch2 := origin[j-1]
			temp := 0
			if ch1 != ch2 {
				temp = 1
			}

			matrix[i*(n+1)+j] = minOfThree(matrix[(i-1)*(n+1)+j]+1, matrix[i*(n+1)+j-1]+1, matrix[(i-1)*(n+1)+j-1]+temp)
		}
	}

	percent := 1 - float64(matrix[(m+1)*(n+1)-1])/float64(max(m, n))
	log.Printf("NameMatch %v %v Percent:%v", match, origin, percent)

	if percent < standard {
		return false
	}

	return true

}

/*生成二维数组*/
func newMatrix(m, n int) []int {
	var ret []int
	ret = make([]int, (m+1)*(n+1))

	for i := 0; i <= n; i++ {
		ret[i] = i
	}

	for i := 0; i <= m; i++ {
		ret[i*(n+1)] = i
	}
	return ret
}

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func minOfThree(a, b, c int) int {
	tmp := min(a, b)
	return min(tmp, c)
}
