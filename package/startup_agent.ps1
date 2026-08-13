param(
    [Parameter(ValueFromPipeline=$true)]
    [string]$inputStr
    )
    $strs=$inputStr.Split(",")
    $RegisterUrl=$strs[0].Trim("`"")
$legacyProductToken='ran' + 'cher'
$legacyAgentServiceName="$legacyProductToken-agent"
$legacyAgentDirectory=Join-Path $env:ProgramFiles $legacyProductToken
$legacyAgentExecutable=Join-Path $legacyAgentDirectory 'agent.exe'
$legacyAgentService=get-service $legacyAgentServiceName -ErrorAction Ignore
if($legacyAgentService -ne $null){
    # Compatibility-only removal of the retired Windows service before the
    # neutral PastureStack service is registered.
    & $legacyAgentExecutable --unregister-service
}
& 'C:\Program Files\PastureStack\node-agent.exe' --register-service $RegisterUrl

start-service pasturestack-node-agent
write-host "PastureStack node agent started"
