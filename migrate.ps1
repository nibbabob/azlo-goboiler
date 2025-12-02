# Create-Secrets.ps1

# --- Configuration ---
$envFile = ".env"
$secretsDir = ".\secrets"
# --- End Configuration ---

if (-not (Test-Path $envFile)) {
    Write-Error "Error: The '$envFile' file was not found in the current directory."
    exit 1
}

if (-not (Test-Path $secretsDir)) {
    Write-Host "Creating secrets directory at '$secretsDir'..."
    New-Item -ItemType Directory -Path $secretsDir | Out-Null
}

$envVars = @{}

# --- Pass 1: Read all variables ---
Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -and !$line.StartsWith("#")) {
        $separatorIndex = $line.IndexOf("=")
        if ($separatorIndex -ne -1) {
            $key = $line.Substring(0, $separatorIndex).Trim()
            $value = $line.Substring($separatorIndex + 1).Trim()
            if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
                $value = $value.Substring(1, $value.Length - 2)
            }
            $envVars[$key] = $value
        }
    }
}

# --- Pass 2: Resolve nested variables ---
$resolvedVars = $envVars.Clone()
foreach ($key in $envVars.Keys) {
    $value = $envVars[$key]
    $matches = [regex]::Matches($value, '\$\{?(\w+)\}?')
    foreach ($match in $matches) {
        $varToReplace = $match.Groups[1].Value
        if ($envVars.ContainsKey($varToReplace)) {
            $value = $value.Replace($match.Value, $envVars[$varToReplace])
        }
    }
    $resolvedVars[$key] = $value
}

# --- Pass 3: Write the resolved variables to secret files ---
Write-Host "Writing secrets to files..."
foreach ($key in $resolvedVars.Keys) {
    $secretFileName = "$($key.ToLower()).txt"
    $secretFilePath = Join-Path $secretsDir $secretFileName
    
    # MODIFIED: Use a .NET method that writes UTF-8 without a BOM by default.
    [System.IO.File]::WriteAllText($secretFilePath, $resolvedVars[$key])
    
    Write-Host "  - Created '$secretFilePath'"
}

Write-Host "Secret files created successfully."