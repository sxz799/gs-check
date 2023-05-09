package check

import (
	"bytes"
	"fmt"
	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
	"gsCheck/model"
	"gsCheck/utils"
	"io"
	"strconv"
	"strings"
)

func PreCheck(fileType string, r io.Reader) (num int, errs []model.ErrInfo) {
	rows := make([][]string, 0)
	switch fileType {
	case "xlsx":
		excelFile, err := excelize.OpenReader(r)
		if err != nil {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: err.Error(),
				FixMsg:   "文件读取失败,请按照使用说明导出格式为xlsx的文件进行检测",
			})
			return
		} else {
			fmt.Println("文件读取成功！")
		}
		count := excelFile.SheetCount
		if count != 1 {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: "sheet工作表太多",
				FixMsg:   "请仅保留一个工作表,当前文件有" + strconv.Itoa(count) + "个sheet表(如果仍提示此错误,请右键点击左下角现在的工作表并点击`取消隐藏工作表`)",
			})
		}
		sheets := excelFile.GetSheetList()
		rows, _ = excelFile.GetRows(sheets[0])
		if len(rows) < 4 {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: "格式不正确",
				FixMsg:   "至少在第4行要有数据",
			})
			return
		}
	case "xls":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r) // 将 io.Reader 对象读取到缓冲区中
		if err != nil {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: err.Error(),
				FixMsg:   "xls文件读取失败,请按照使用说明导出格式为xlsx的文件进行检测",
			})
			return
		}
		reader := bytes.NewReader(buf.Bytes())
		excelFile, err := xls.OpenReader(reader, "utf-8")
		count := excelFile.NumSheets()
		if count != 1 {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: "sheet工作表太多",
				FixMsg:   "请仅保留一个工作表,当前文件有" + strconv.Itoa(count) + "个sheet表(如果仍提示此错误,请右键点击左下角现在的工作表并点击`取消隐藏工作表`)",
			})
		}
		sheet := excelFile.GetSheet(0)
		maxRow := sheet.MaxRow
		if maxRow < 4 {
			errs = append(errs, model.ErrInfo{
				ErrorMsg: "格式不正确",
				FixMsg:   "至少在第4行要有数据",
			})
			return
		}
		row := sheet.Row(3)
		lastCol := row.LastCol()
		for strings.ReplaceAll(row.Col(lastCol), " ", "") == "" {
			lastCol--
		}

		for i := 0; i < int(maxRow); i++ {
			row = sheet.Row(i)
			trow := make([]string, 0)
			for j := 0; j < lastCol; j++ {
				trow = append(trow, row.Col(j))
			}
			rows = append(rows, trow)
		}
	}

	rows = rows[2:]
	cellA3 := rows[0][0]
	cellA3 = strings.ReplaceAll(cellA3, " ", "")
	if cellA3 == "" || !strings.Contains("资产编号,资产名称,资产来源,管理类别,类别名称1,资产状态,是否计提折旧,入账日期,资产原值,累计折旧,折旧方法,资产数量,净残值率(%),净残值,月折旧率(%),月折旧额,年折旧率(%),年折旧额,存放地点,部门名称,责任人,入账时累计折旧,减值准备,已提月份,未计提月份,单位名称,使用部门,使用人,使用月份,计量单位,备注,实际数量", cellA3) {
		errs = append(errs, model.ErrInfo{
			ErrorMsg: "表格结构错误",
			FixMsg:   "请将前两行内容清空并合并为1个单元格!(资产编号,资产名称,资产来源等标题要在第三行!)",
		})
		return
	}

	n, errs2 := check(rows)
	errs = append(errs, errs2...)
	num = num + n
	return
}

func check(rows [][]string) (num int, errs []model.ErrInfo) {

	//列号和标题的map
	indexTitleMap := make(map[int]string)
	//所有的标题
	titles := make([]string, 0)

	//标题行
	titleRow := rows[0]
	//遍历标题行 记录列号和标题的关系
	for i, cell := range titleRow {
		if len(cell) > 0 {
			indexTitleMap[i] = cell
			titles = append(titles, cell)
		}
	}

	//todo 判断是否缺失某一列

	rows = rows[1:]
	//gsid的map 用于判断是否存在相同的gsid
	var GsIdMp = make(map[string]string)
	for index, row := range rows {
		if strings.Contains(row[0], "合计") {
			break
		}

		lastErrorMktName := ""
		go func(row []string) {
			titleValueMap := make(map[string]string)
			for k, cell := range row {
				title := indexTitleMap[k]
				titleValueMap[title] = cell
				f, ok := utils.TitleCheckFuncMap[title]
				if ok {
					correct, errInfo := f(cell)
					if !correct {
						errInfo.Line = index + 4
						errInfo.ErrorMsg = title + errInfo.ErrorMsg
						errs = append(errs, errInfo)
					}
				}
			}
			GsId := titleValueMap["资产编号"]
			if len(strings.ReplaceAll(GsId, " ", "")) < 1 {
				errs = append(errs, model.ErrInfo{
					Line:     index + 4,
					ErrorMsg: "资产编号错误",
					FixMsg:   "资产编号不可为空",
				})
			} else {
				_, ok := GsIdMp[GsId]
				if ok {
					errs = append(errs, model.ErrInfo{
						Line:     index + 4,
						ErrorMsg: "资产编号:" + GsId + "已存在",
						FixMsg:   "修改为不重复的编码(比如后面加上一些字母)",
					})
				} else {
					GsIdMp[GsId] = GsId
				}
			}

			tNum := titleValueMap["资产数量"]
			if strings.Contains(tNum, ".00") {
				tNum = strings.ReplaceAll(tNum, ".00", "")
			}
			tNum2 := titleValueMap["实际数量"]
			if strings.Contains(tNum2, ".00") {
				tNum2 = strings.ReplaceAll(tNum2, ".00", "")
			}
			capNum, _ := strconv.Atoi(tNum)
			capRealNum, _ := strconv.Atoi(tNum2)

			if capNum < 1 {
				errs = append(errs, model.ErrInfo{
					Line:     index + 4,
					ErrorMsg: "资产数量异常",
					FixMsg:   "资产数量不可小于1",
				})
			}
			if capRealNum < 1 {
				errs = append(errs, model.ErrInfo{
					Line:     index + 4,
					ErrorMsg: "实际数量异常",
					FixMsg:   "实际数量不可小于1",
				})
			}

			if capNum > 0 && capRealNum > 0 {
				errs = append(errs, model.ErrInfo{
					Line:     index + 4,
					ErrorMsg: "资产数量和实际数量关系异常",
					FixMsg:   "资产数量和实际数量不可同时大于1",
				})
			}

			mkt, ok := titleValueMap["单位名称"]
			if ok {
				correct, errInfo := IsCorrectMKT(mkt)
				if !correct && mkt != lastErrorMktName {
					lastErrorMktName = mkt
					errInfo.Line = index + 4
					errInfo.ErrorMsg = "单位名称" + errInfo.ErrorMsg
					errInfo.FixMsg = errInfo.FixMsg + "(单位名称错误只记录一条,但所有的记录都要修改)"
					errs = append(errs, errInfo)
				} else {
					correct, errInfo = IsCorrectDept(titleValueMap["部门名称"], mkt)
					if !correct {
						errInfo.Line = index + 4
						errInfo.ErrorMsg = "部门名称" + errInfo.ErrorMsg
						errs = append(errs, errInfo)
					}
					correct, errInfo = IsCorrectDept(titleValueMap["使用部门"], mkt)
					if !correct {
						errInfo.Line = index + 4
						errInfo.ErrorMsg = "使用部门" + errInfo.ErrorMsg
						errs = append(errs, errInfo)
					}

					users, ok := titleValueMap["责任人"]
					if ok {
						if strings.Contains(users, "+") {
							if capNum > 0 && capNum != len(strings.Split(users, "+")) {
								errs = append(errs, model.ErrInfo{
									Line:     index + 4,
									ErrorMsg: "责任人数量配置异常！",
									FixMsg:   "修改资产数量与使用人的数量(资产数量大于1时,人员要么1个，要么与资产数量相同)",
								})
							}
							for _, user := range strings.Split(users, "+") {
								correct, errInfo = IsCorrectUser(user, mkt)
								if !correct {
									errInfo.Line = index + 4
									errInfo.ErrorMsg = "责任人" + errInfo.ErrorMsg
									errs = append(errs, errInfo)
								}
							}
						} else {
							correct, errInfo = IsCorrectUser(users, mkt)
							if !correct {
								if !correct {
									errInfo.Line = index + 4
									errInfo.ErrorMsg = "责任人" + errInfo.ErrorMsg
									errs = append(errs, errInfo)
								}
							}
						}

					}

					users2, ok := titleValueMap["使用人"]
					if ok {
						if strings.Contains(users2, "+") {
							if capNum > 0 && capNum != len(strings.Split(users2, "+")) {
								errs = append(errs, model.ErrInfo{
									Line:     index + 4,
									ErrorMsg: "使用人数量配置异常！",
									FixMsg:   "修改资产数量与使用人的数量(资产数量大于1时,人员要么1个，要么与资产数量相同)",
								})
							}
							for _, user := range strings.Split(users2, "+") {
								correct, errInfo = IsCorrectUser(user, mkt)
								if !correct {
									if !correct {
										errInfo.Line = index + 4
										errInfo.ErrorMsg = "使用人" + errInfo.ErrorMsg
										errs = append(errs, errInfo)
									}
								}
							}
						} else {
							correct, errInfo = IsCorrectUser(users2, mkt)
							if !correct {
								if !correct {
									errInfo.Line = index + 4
									errInfo.ErrorMsg = "使用人" + errInfo.ErrorMsg
									errs = append(errs, errInfo)
								}
							}
						}

					}
				}

			}

			if titleValueMap["是否计提折旧"] == "是" {
				syyf, _ := strconv.Atoi(titleValueMap["使用月份"])
				ytyf, _ := strconv.Atoi(titleValueMap["已提月份"])
				wtyf, _ := strconv.Atoi(titleValueMap["未计提月份"])

				if wtyf < 1 {
					errs = append(errs, model.ErrInfo{
						Line:     index + 4,
						ErrorMsg: "未计提月份异常",
						FixMsg:   "计提折旧时未计提月份不可小于1",
					})
				}
				if syyf != ytyf+wtyf {
					errs = append(errs, model.ErrInfo{
						Line:     index + 4,
						ErrorMsg: "使用月份不等于已提月份加未提月份",
						FixMsg:   "校对使用月份，已提月份，未提月份",
					})
				}
				float, _ := strconv.ParseFloat(titleValueMap["净残值率(%)"], 64)
				float = float / 100
				if float == 1 {
					errs = append(errs, model.ErrInfo{
						Line:     index + 4,
						ErrorMsg: "净残值率错误",
						FixMsg:   "净残值率在计提时不可等于100%",
					})
				}
			}
		}(row)

	}
	num = len(rows)
	return
}
