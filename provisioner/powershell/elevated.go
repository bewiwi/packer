package powershell

import (
	"text/template"
)

type elevatedOptions struct {
	User              string
	Password          string
	TaskName          string
	TaskDescription   string
	LogFile           string
	XMLEscapedCommand string
}

var elevatedTemplate = template.Must(template.New("ElevatedCommand").Parse(`
$name = "{{.TaskName}}"
$log = [System.Environment]::ExpandEnvironmentVariables("{{.LogFile}}")
$s = New-Object -ComObject "Schedule.Service"
$s.Connect()
$t = $s.NewTask($null)
$xml = [xml]@'
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>{{.TaskDescription}}</Description>
  </RegistrationInfo>
  <Principals>
    <Principal id="Author">
      <UserId>{{.User}}</UserId>
      <LogonType>Password</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT24H</ExecutionTimeLimit>
    <Priority>4</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>cmd</Command>
      <Arguments>/c {{.XMLEscapedCommand}}</Arguments>
    </Exec>
  </Actions>
</Task>
'@
$logon_type = 1
$password = "{{.Password}}"
if ($password.Length -eq 0) {
  $logon_type = 5
  $password = $null
  $ns = New-Object System.Xml.XmlNamespaceManager($xml.NameTable)
  $ns.AddNamespace("ns", $xml.DocumentElement.NamespaceURI)
  $node = $xml.SelectSingleNode("/ns:Task/ns:Principals/ns:Principal/ns:LogonType", $ns)
  $node.ParentNode.RemoveChild($node) | Out-Null
}
$t.XmlText = $xml.OuterXml
if (Test-Path variable:global:ProgressPreference){$ProgressPreference="SilentlyContinue"}
$f = $s.GetFolder("\")
$f.RegisterTaskDefinition($name, $t, 6, "{{.User}}", $password, $logon_type, $null) | Out-Null
$t = $f.GetTask("\$name")
$t.Run($null) | Out-Null
$timeout = 10
$sec = 0
while ((!($t.state -eq 4)) -and ($sec -lt $timeout)) {
  Start-Sleep -s 1
  $sec++
}

$line = 0
do {
  Start-Sleep -m 100
  if (Test-Path $log) {
    Get-Content $log | select -skip $line | ForEach {
      $line += 1
      Write-Output "$_"
    }
  }
} while (!($t.state -eq 3))
$result = $t.LastTaskResult
if (Test-Path $log) {
    Remove-Item $log -Force -ErrorAction SilentlyContinue | Out-Null
}
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($s) | Out-Null
exit $result`))
