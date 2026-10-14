param([Parameter(Mandatory=$true)][string]$Path)
$ErrorActionPreference = 'Stop'
if (!('Pomme.NativeSourcePaths' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Text;
using System.Text.RegularExpressions;
namespace Pomme {
    public static class NativeSourcePaths {
        public static int Scrub(string path, string profile) {
            byte[] data = File.ReadAllBytes(path);
            int count = 0;
            foreach (Encoding encoding in new Encoding[] { Encoding.UTF8, Encoding.Unicode }) {
                int width = encoding == Encoding.Unicode ? 2 : 1;
                foreach (string prefix in new string[] { profile, profile.Replace('\\', '/') }) {
                    byte[] needle = encoding.GetBytes(prefix);
                    byte[] replacement = encoding.GetBytes(new string('_', needle.Length / width));
                    for (int i = 0; i <= data.Length - needle.Length; i++) {
                        bool match = true;
                        for (int j = 0; j < needle.Length; j++) {
                            byte a = data[i + j], b = needle[j];
                            if (a >= 97 && a <= 122) a -= 32;
                            if (b >= 97 && b <= 122) b -= 32;
                            if (a != b) { match = false; break; }
                        }
                        if (!match) continue;
                        int end = i;
                        while (end + width <= data.Length && end - i < 4096) {
                            if (data[end] == 0 && (width == 1 || data[end + 1] == 0)) break;
                            end += width;
                        }
                        string value = encoding.GetString(data, i, end - i).Replace('\\', '/');
                        // Only compiler source-filename literals in dependency caches
                        // may be replaced. Unexpected paths abort the release.
                        if (!Regex.IsMatch(value, @"/(?:\.cargo|CARGO~[0-9]+)/.*\.(?:c|h|cc|cpp|hpp|rs)$", RegexOptions.IgnoreCase))
                            throw new InvalidOperationException("Unexpected private path in binary; inspect before publishing.");
                        Buffer.BlockCopy(replacement, 0, data, i, replacement.Length);
                        count++;
                        i += needle.Length - 1;
                    }
                }
            }
            File.WriteAllBytes(path, data);
            return count;
        }
    }
}
'@
}
$profileRoot = [Environment]::GetFolderPath('UserProfile')
$count = [Pomme.NativeSourcePaths]::Scrub([IO.Path]::GetFullPath($Path), $profileRoot)
Write-Host "Anonymized $count native dependency source-filename records; executable length and offsets retained."
