package main

import (
	"crypto/rand"
	"fmt"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/ssh"
	"log"
	"strings"
	"sync"
)

func generateRandomIPv6() string {
	ip := make([]byte, 16)

	// Đặt các giá trị cố định để đảm bảo định dạng đúng của IPv6
	ip[0] = 0x20
	ip[1] = 0x01

	// Tạo số ngẫu nhiên cho 12 byte cuối
	_, err := rand.Read(ip[8:])
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("2001:19f0:7001:321a:%02x:%02x:%02x:%02x", ip[8], ip[9], ip[10], ip[11])
}
func main() {
	// Thông tin SSH
	sshHost := "108.61.250.38"
	sshPort := 22
	sshUser := "root"
	sshPassword := "9nB+M7f%Z4]nT@2Q"
	fileExcel, _ := excelize.OpenFile("./data2.xlsx")
	filePath := "/etc/squid/squid.conf"
	// Tạo kết nối SSH
	sshConfig := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Thực hiện kết nối SSH
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshHost, sshPort), sshConfig)
	if err != nil {
		fmt.Println("Lỗi khi kết nối SSH:", err)
		return
	}
	defer sshClient.Close()
	session, err := sshClient.NewSession()
	if err != nil {
		fmt.Println("Lỗi khi tạo phiên SSH:", err)
		return
	}
	cmd := fmt.Sprintf("cat %s", filePath)
	content, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Println("Lỗi khi đọc file:", err)
		return
	}
	defer session.Close()

	// Chuyển nội dung file thành chuỗi
	fileString := string(content)
	index := strings.Index(fileString, "#tcpoutgoingaddress")
	if index == -1 {
		fmt.Println("Không tìm thấy dòng chứa '#httpport' trong file.")
		return
	}
	ipv6Remove := []string{}
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for i := 1; i <= 300; i++ {
		wg.Add(1)
		go func(i int) {
			session1, err := sshClient.NewSession()
			if err != nil {
				fmt.Println("Lỗi khi tạo phiên SSH:", err)
				return
			}
			ipv6, _ := fileExcel.GetCellValue("Sheet1", fmt.Sprintf("E%d", i))
			ipv6Gen := generateRandomIPv6()
			mutex.Lock()
			ipv6Remove = append(ipv6Remove, ipv6)
			fileString = strings.ReplaceAll(fileString, ipv6, ipv6Gen)
			mutex.Unlock()
			cmd := fmt.Sprintf("sudo ip -6 address add %s/64 dev enp1s0", ipv6Gen)

			//cmd := fmt.Sprintf("sudo ufw allow %d", 24000+i)
			//fmt.Println(cmd)
			output, err := session1.CombinedOutput(cmd)
			if err != nil {
				fmt.Println(string(output))
				log.Fatalf("Lỗi khi thực thi lệnh: %v", err)
			}
			fmt.Println(output)
			session1.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	cmd = fmt.Sprintf("echo \"%s\" > %s", fileString, filePath)
	_, err = session.CombinedOutput(cmd)
	if err != nil {
		fmt.Println("Lỗi khi ghi file:", err)
		return
	}
	output, err := session.CombinedOutput("sudo service squid restart")
	if err != nil {
		fmt.Println(string(output))
		log.Fatalf("Lỗi khi thực thi lệnh: %v", err)
	}
	for _, ip := range ipv6Remove {
		wg.Add(1)
		go func(ip string) {
			session, err := sshClient.NewSession()
			if err != nil {
				fmt.Println("Lỗi khi tạo phiên SSH:", err)
				return
			}
			cmd := fmt.Sprintf("sudo ip -6 address del %s/64 dev enp1s0", ip)
			output, err := session.CombinedOutput(cmd)
			if err != nil {
				fmt.Println(string(output))
				log.Fatalf("Lỗi khi thực thi lệnh: %v", err)
			}
			fmt.Println(output)
			session.Close()
		}(ip)
	}

	fmt.Println("Đã thêm địa chỉ IPv6 thành công!")

}
