param(
 [Parameter(Mandatory=$true)][string]$RunUrl,
 [string]$OutputPath="evidence.json",
 [string]$JobName="",
 [int]$MaxLogs=3
)
$ErrorActionPreference="Stop"
$u=[uri]$RunUrl
if($u.Scheme -ne "https" -or $u.Host -ne "github.com" -or $u.AbsolutePath -notmatch '^/([\w.-]+)/([\w.-]+)/actions/runs/(\d+)(?:/attempts/(\d+))?/?$'){throw "Invalid run URL"}
$repo="$($Matches[1])/$($Matches[2])";$runId=$Matches[3];$attempt=$Matches[4]
function Get-GitHubJson([string]$Path) {
 $data=& gh api $Path
 if($LASTEXITCODE -ne 0){throw "GitHub read failed: $Path"}
 return ($data -join "`n" | ConvertFrom-Json)
}
$path="repos/$repo/actions/runs/$runId"
if($attempt){$path+="/attempts/$attempt"}
$run=Get-GitHubJson $path
if($run.repository.private){throw "Public repositories only"}
$jobs=@()
for($page=1;$page -le 100;$page++){
 $part=Get-GitHubJson "repos/$repo/actions/runs/$runId/attempts/$($run.run_attempt)/jobs?per_page=100&page=$page"
 $jobs+=@($part.jobs)
 if($part.jobs.Count -lt 100){break}
}
$files=@{};$logs=@{};$workflowPath=($run.path -split '@')[0]
function Save-Workflow([string]$RepoName,[string]$Sha) {
 $key="$RepoName@$($Sha):$workflowPath"
 if($files.ContainsKey($key)){return}
 $v=Get-GitHubJson "repos/$RepoName/contents/$($workflowPath)?ref=$Sha"
 $files[$key]=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($v.content))
}
Save-Workflow $repo $run.head_sha
$selected=@($jobs | Where-Object {$_.conclusion -eq "failure" -and $_.labels -contains "ubuntu-latest" -and (!$JobName -or $_.name -eq $JobName -or "$($_.id)" -eq $JobName)} | Sort-Object id | Select-Object -First $MaxLogs)
foreach($j in $selected){
 $endpoint="/repos/$repo/actions/jobs/$($j.id)/logs"
 $lines=& gh api $endpoint --allow-escape-sequences
 if($LASTEXITCODE -ne 0){Write-Warning "Logs unavailable for $($j.name)";continue}
 $log=$lines -join "`n";$logs[$endpoint]=$log
 $m=[regex]::Match($log,'git log -1 --format=.?(?:%H).?\r?\n\S+ ([a-f0-9]{40})')
 if($m.Success){Save-Workflow $repo $m.Groups[1].Value}
}
$bundle=@{url=$RunUrl;run=$run;jobs=@($jobs);files=$files;logs=$logs}
$parent=Split-Path -Parent $OutputPath
if($parent){New-Item -ItemType Directory -Force -Path $parent | Out-Null}
$bundle | ConvertTo-Json -Depth 100 | Set-Content -LiteralPath $OutputPath -Encoding utf8NoBOM
Write-Host "Exported public evidence: $OutputPath ($($logs.Count) job logs; no credentials included)"
