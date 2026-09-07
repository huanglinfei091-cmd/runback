param(
 [ValidateSet("ToUbuntu","ToWindows")][string]$Direction="ToUbuntu",
 [Parameter(Mandatory=$true)][string]$Ubuntu,
 [string]$UbuntuRoot="runback-work/runback",
 [string]$WindowsRoot=(Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)
$ErrorActionPreference="Stop"
# Copy only. Never mirror/delete and never transfer .git, credentials or replay workspaces.
$items=@("cmd","internal","pkg","docs","examples","testdata","scripts",".github","benchmark","go.mod","go.sum","README.md","README.zh-CN.md","LICENSE",".gitignore",".gitattributes","AGENTS.md","CONTRIBUTING.md","SECURITY.md","action.yml")
if($Direction -eq "ToUbuntu"){
 & ssh $Ubuntu "mkdir -p '$UbuntuRoot'"
 foreach($item in $items){
  $path=Join-Path $WindowsRoot $item
  if(Test-Path -LiteralPath $path){& scp -r $path "$($Ubuntu):$UbuntuRoot/";if($LASTEXITCODE){throw "Copy failed: $item"}}
 }
}else{
 foreach($item in $items){
  & ssh $Ubuntu "test -e '$UbuntuRoot/$item'"
  if($LASTEXITCODE -eq 0){& scp -r "$($Ubuntu):$UbuntuRoot/$item" $WindowsRoot;if($LASTEXITCODE){throw "Copy failed: $item"}}
 }
}
Write-Host "Copied $Direction. Deletions and conflicting edits must be reviewed with git diff."
