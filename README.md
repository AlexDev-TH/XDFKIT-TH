![XDFKIT-TH](app/branding/banner.png)

# XDFKIT-TH

โปรแกรมแปลงไฟล์นิยามแมพ ECU สำหรับคนไทย ใช้งานผ่านหน้าต่าง ไม่ต้องพิมพ์คำสั่ง
แปลงไปมาได้ระหว่าง WinOLS KP, TunerPro XDF, A2L, DAMOS, JSON และ YAML
พร้อมดูค่าแมพจริงจากไฟล์ BIN และตรวจแก้แกนของไฟล์ KP

พัฒนาโดย **AlexDev-TH** ต่อยอดจากเอนจิน [xdfkit ของ nyetlabs](https://github.com/nyetlabs/xdfkit)

## ดาวน์โหลด

ไปที่หน้า [Releases](../../releases) แล้วโหลด `XDFKIT-TH.exe` มาเปิดได้เลย ไม่ต้องติดตั้ง

- ใช้ได้บน Windows 10 และ 11 แบบ 64 บิต
- ต้องมี WebView2 Runtime ซึ่ง Windows 11 มีมาให้แล้ว
- ไฟล์ยังไม่ได้เซ็นรับรอง ครั้งแรก Windows อาจขึ้นหน้าต่าง SmartScreen ให้กด "ข้อมูลเพิ่มเติม" แล้ว "เรียกใช้ต่อไป"

## ทำอะไรได้บ้าง

| หน้า | ใช้ทำอะไร |
| --- | --- |
| แปลงไฟล์ | ลากไฟล์นิยามแมพมาวาง เลือกรูปแบบปลายทาง แล้วกดแปลง แปลงทีละหลายไฟล์ได้ |
| สำรวจแมพ | ดูรายการแมพทั้งหมด ค้นหา กรองตามชนิดและหมวด ดูค่าจริงจากไฟล์ BIN เป็นตารางไล่เฉดสี กราฟ และพื้นผิว 3 มิติ |
| ตรวจ & แก้ KP | เทียบไฟล์ KP กับไฟล์ BIN หาแกนที่ตั้งค่าผิด แล้วเขียนไฟล์ที่แก้แล้วเป็นไฟล์ใหม่ |
| รวมไฟล์ | รวมไฟล์นิยามสองไฟล์เข้าด้วยกันโดยจับคู่จากชื่อแมพ ดูผลก่อนได้ |
| ตรวจ Stamp | ตรวจว่าไฟล์ JSON หรือ YAML ถูกแก้ด้วยมือหลังจากโปรแกรมเขียนหรือไม่ |

ตัวโปรแกรมสลับภาษาไทยกับอังกฤษได้ มีธีมมืด ธีมสว่าง และตามระบบ

### รูปแบบไฟล์

| รูปแบบ | อ่าน | เขียน |
| --- | --- | --- |
| WinOLS KP | ได้ | ได้ |
| TunerPro XDF | ได้ | ได้ |
| A2L | ได้ | |
| DAMOS (.dam) | ได้ | |
| JSON, YAML | ได้ | ได้ |
| CSV รายการแมพ | | ได้ |

### ข้อความภาษาไทยในไฟล์ KP และ XDF

WinOLS และ TunerPro บน Windows ภาษาไทยเก็บข้อความไทยเป็นรหัส Windows-874
โปรแกรมนี้อ่านข้อความแบบนั้นออกมาเป็นภาษาไทยจริง และเขียนกลับในแบบเดียวกัน
ชื่อแมพและคำอธิบายภาษาไทยจึงข้ามระหว่าง KP กับ XDF ได้ครบทุกตัวอักษร
ส่วนไฟล์ JSON และ YAML จะได้ข้อความไทยแบบ UTF-8 ที่เปิดแก้ด้วยโปรแกรมแก้ข้อความทั่วไปได้

ข้อความที่ไม่ใช่ภาษาไทยจะถูกส่งผ่านเอนจินตามเดิมโดยไม่แตะต้อง

### สิ่งที่ควรรู้

- ไฟล์ต้นฉบับจะไม่ถูกเขียนทับ ถ้าปลายทางมีไฟล์ชื่อเดียวกันอยู่ โปรแกรมจะหยุดจนกว่าจะเปิดตัวเลือก "เขียนทับไฟล์ที่มีอยู่"
- แปลงเป็น XDF จะได้ไฟล์ `.meta.json` คู่กันมาด้วย เก็บไว้ข้างไฟล์ XDF จะแปลงกลับเป็น KP ได้เหมือนเดิม
- ไฟล์ KP บางรุ่นเอนจินยังอ่านไม่ได้ โปรแกรมจะแจ้งข้อความจากเอนจินให้เห็นตรง ๆ
- การตรวจแกน KP รองรับ ECU ตระกูล Bosch ME7 และ Motronic M3/M5 ตามที่เอนจินรองรับ

## สร้างจากซอร์ส

ต้องมี Go 1.26.4 ขึ้นไป, Node.js 20.19 ขึ้นไป และ Wails CLI รุ่น 2

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd app
powershell -File build.ps1
```

ได้ไฟล์ `app\build\bin\XDFKIT-TH.exe`

ทดสอบ

```powershell
go test ./...          # เอนจิน
cd app
go test ./...          # ส่วนเชื่อมหน้าต่างกับเอนจิน และข้อความภาษาไทย
```

เทสต์ในโฟลเดอร์ `app` ต้องมีโฟลเดอร์ `app\frontend\dist` ซึ่งได้มาจากการ build หนึ่งครั้ง

## โครงสร้าง

| โฟลเดอร์ | เนื้อหา |
| --- | --- |
| `app/` | โปรแกรมหน้าต่าง เขียนด้วย Go กับ Wails ส่วนหน้าจอเป็น React กับ TypeScript |
| `app/branding/` | โลโก้ แบนเนอร์ และสคริปต์สร้างไอคอน |
| `api/`, `kp/`, `xdf/`, `a2l/`, `dam/`, `model/`, `lint/`, `canon/`, `mapcsv/` | เอนจิน xdfkit ต้นฉบับ ไม่ได้แก้ไข |
| `cmd/xdfkit/` | โปรแกรมบรรทัดคำสั่งของต้นฉบับ ยังใช้ได้ตามปกติ |
| `docs/` | เอกสารของเอนจินต้นฉบับ ภาษาอังกฤษ |

## เครดิตและสัญญาอนุญาต

- XDFKIT-TH โดย AlexDev-TH
- เอนจิน xdfkit โดย nyetlabs
- ฟอนต์ IBM Plex Sans Thai, IBM Plex Mono และ Kanit ใช้สัญญาอนุญาต SIL Open Font License
- ไอคอนจาก Lucide ใช้สัญญาอนุญาต ISC

เผยแพร่ภายใต้ LGPL-3.0-or-later เช่นเดียวกับต้นฉบับ ดูไฟล์ `COPYING.LESSER` และ `COPYING`

## English

XDFKIT-TH is a Windows desktop app for converting ECU map definitions between WinOLS KP, TunerPro XDF, A2L, DAMOS, JSON and YAML. It shows map values read from a flash image, checks and fixes KP axes, merges definitions, and reads and writes Thai text (Windows-874) in KP and XDF files. The interface is in Thai and English, with dark and light themes.

It is built by AlexDev-TH on the unmodified [xdfkit](https://github.com/nyetlabs/xdfkit) engine by nyetlabs, under the same licence, LGPL-3.0-or-later. Download `XDFKIT-TH.exe` from the Releases page, or build it with `app/build.ps1`.
