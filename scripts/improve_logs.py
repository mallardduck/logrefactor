#!/usr/bin/env python3
"""
AI-Assisted Log Message Improvement Script (Multi-Provider)

Supports: Claude (Anthropic), Gemini (Google), GitHub Copilot

Usage:
    # Claude (Anthropic)
    export ANTHROPIC_API_KEY="your-key"
    python improve_logs.py input.csv output.csv --provider claude

    # Gemini (Google)
    export GOOGLE_API_KEY="your-key"
    python improve_logs.py input.csv output.csv --provider gemini

    # GitHub Copilot
    export GITHUB_TOKEN="your-token"
    python improve_logs.py input.csv output.csv --provider copilot
"""

import sys
import os
import argparse
import pandas as pd
from typing import Optional

# Provider-specific imports
try:
    from anthropic import Anthropic
    ANTHROPIC_AVAILABLE = True
except ImportError:
    ANTHROPIC_AVAILABLE = False

try:
    import google.generativeai as genai
    GEMINI_AVAILABLE = True
except ImportError:
    GEMINI_AVAILABLE = False

try:
    import requests
    REQUESTS_AVAILABLE = True
except ImportError:
    REQUESTS_AVAILABLE = False


class LogImprover:
    """Base class for log improvement providers"""
    
    def __init__(self, api_key: str):
        self.api_key = api_key
    
    def build_prompt(self, original_text: str, function_call: str, arguments: str, log_level: str) -> str:
        """Build the improvement prompt"""
        clean_text = original_text.strip('"\'`')
        
        context = f"Function: {function_call}\nLog Level: {log_level}"
        if arguments:
            context += f"\nArguments: {arguments}"
        
        prompt = f"""Improve this Go log message for clarity, professionalism, and consistency.

{context}
Original message: {clean_text}

Requirements:
- Use clear, professional language
- Be specific and actionable
- Use sentence case (capitalize first word only, except proper nouns)
- If this is a format string with %v, %s, %d, etc., keep those format verbs
- Return ONLY the improved message text, without quotes
- Keep it concise (under 100 characters if possible)

Improved message:"""
        
        return prompt
    
    def improve_message(self, original_text: str, function_call: str, arguments: str, log_level: str) -> Optional[str]:
        """Improve a single log message - to be implemented by subclasses"""
        raise NotImplementedError


class ClaudeImprover(LogImprover):
    """Claude (Anthropic) provider"""
    
    def __init__(self, api_key: str):
        super().__init__(api_key)
        if not ANTHROPIC_AVAILABLE:
            raise ImportError("anthropic package not installed. Run: pip install anthropic")
        self.client = Anthropic(api_key=api_key)
    
    def improve_message(self, original_text: str, function_call: str, arguments: str, log_level: str) -> Optional[str]:
        try:
            prompt = self.build_prompt(original_text, function_call, arguments, log_level)
            
            message = self.client.messages.create(
                model="claude-sonnet-4-20250514",
                max_tokens=300,
                messages=[{"role": "user", "content": prompt}]
            )
            
            improved = message.content[0].text.strip()
            improved = improved.strip('"\'`')
            return improved
            
        except Exception as e:
            print(f"Error with Claude API: {e}")
            return None


class GeminiImprover(LogImprover):
    """Gemini (Google) provider"""
    
    def __init__(self, api_key: str):
        super().__init__(api_key)
        if not GEMINI_AVAILABLE:
            raise ImportError("google-generativeai package not installed. Run: pip install google-generativeai")
        genai.configure(api_key=api_key)
        self.model = genai.GenerativeModel('gemini-pro')
    
    def improve_message(self, original_text: str, function_call: str, arguments: str, log_level: str) -> Optional[str]:
        try:
            prompt = self.build_prompt(original_text, function_call, arguments, log_level)
            
            response = self.model.generate_content(prompt)
            
            improved = response.text.strip()
            improved = improved.strip('"\'`')
            return improved
            
        except Exception as e:
            print(f"Error with Gemini API: {e}")
            return None


class CopilotImprover(LogImprover):
    """GitHub Copilot provider"""
    
    def __init__(self, api_key: str):
        super().__init__(api_key)
        if not REQUESTS_AVAILABLE:
            raise ImportError("requests package not installed. Run: pip install requests")
        self.github_token = api_key
    
    def improve_message(self, original_text: str, function_call: str, arguments: str, log_level: str) -> Optional[str]:
        try:
            prompt = self.build_prompt(original_text, function_call, arguments, log_level)
            
            # GitHub Copilot uses chat completions endpoint
            url = "https://api.githubcopilot.com/chat/completions"
            
            headers = {
                "Authorization": f"Bearer {self.github_token}",
                "Content-Type": "application/json",
                "Editor-Version": "vscode/1.85.0",
                "Editor-Plugin-Version": "copilot-chat/0.11.0",
            }
            
            payload = {
                "messages": [
                    {"role": "system", "content": "You are a helpful assistant that improves log messages."},
                    {"role": "user", "content": prompt}
                ],
                "model": "gpt-4",
                "temperature": 0.3,
                "max_tokens": 300
            }
            
            response = requests.post(url, json=payload, headers=headers, timeout=30)
            response.raise_for_status()
            
            data = response.json()
            improved = data['choices'][0]['message']['content'].strip()
            improved = improved.strip('"\'`')
            return improved
            
        except Exception as e:
            print(f"Error with GitHub Copilot API: {e}")
            return None


def get_improver(provider: str) -> LogImprover:
    """Factory function to get the appropriate improver"""
    
    if provider == "claude":
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY environment variable not set")
        return ClaudeImprover(api_key)
    
    elif provider == "gemini":
        api_key = os.environ.get("GOOGLE_API_KEY") or os.environ.get("GEMINI_API_KEY")
        if not api_key:
            raise ValueError("GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set")
        return GeminiImprover(api_key)
    
    elif provider == "copilot":
        api_key = os.environ.get("GITHUB_TOKEN")
        if not api_key:
            raise ValueError("GITHUB_TOKEN environment variable not set")
        return CopilotImprover(api_key)
    
    else:
        raise ValueError(f"Unknown provider: {provider}. Choose: claude, gemini, or copilot")


def main():
    parser = argparse.ArgumentParser(
        description="Improve log messages using AI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Using Claude
  export ANTHROPIC_API_KEY="sk-ant-..."
  python improve_logs.py input.csv output.csv --provider claude
  
  # Using Gemini
  export GOOGLE_API_KEY="AIza..."
  python improve_logs.py input.csv output.csv --provider gemini
  
  # Using GitHub Copilot
  export GITHUB_TOKEN="ghp_..."
  python improve_logs.py input.csv output.csv --provider copilot
        """
    )
    
    parser.add_argument("input_csv", help="Input CSV file")
    parser.add_argument("output_csv", help="Output CSV file")
    parser.add_argument(
        "--provider",
        choices=["claude", "gemini", "copilot"],
        default="claude",
        help="AI provider to use (default: claude)"
    )
    parser.add_argument(
        "--skip-existing",
        action="store_true",
        help="Skip rows that already have NewText"
    )
    
    args = parser.parse_args()
    
    # Read CSV
    print(f"Reading {args.input_csv}...")
    try:
        df = pd.read_csv(args.input_csv)
    except Exception as e:
        print(f"Error reading CSV: {e}")
        sys.exit(1)
    
    # Validate CSV structure
    required_columns = ['ID', 'MessageTemplate', 'NewMessage']
    missing_columns = [col for col in required_columns if col not in df.columns]
    if missing_columns:
        print(f"Error: CSV missing required columns: {missing_columns}")
        sys.exit(1)
    
    # Initialize improver
    print(f"Initializing {args.provider} provider...")
    try:
        improver = get_improver(args.provider)
    except Exception as e:
        print(f"Error initializing provider: {e}")
        sys.exit(1)
    
    # Process each row
    total = len(df)
    improved_count = 0
    skipped_count = 0
    failed_count = 0
    
    print(f"\nProcessing {total} log entries using {args.provider}...")
    print("-" * 80)
    
    try:
        for idx, row in df.iterrows():
            # Skip if already has NewMessage and skip-existing flag is set
            if args.skip_existing and pd.notna(row.get('NewMessage')) and row['NewMessage'] != '':
                skipped_count += 1
                continue

            # Get the original text
            original = row.get('MessageTemplate', '')
            if pd.isna(original) or original == '':
                skipped_count += 1
                continue

            print(f"\n[{idx + 1}/{total}] {row['ID']}")
            print(f"  Original: {original}")

            # Get context
            function_call = row.get('OriginalCall', '') if pd.notna(row.get('OriginalCall')) else ''
            arguments = row.get('ArgumentDetails', '') if pd.notna(row.get('ArgumentDetails')) else ''
            log_level = row.get('LogLevel', '') if pd.notna(row.get('LogLevel')) else ''

            # Improve the message
            improved = improver.improve_message(original, function_call, arguments, log_level)

            if improved:
                df.at[idx, 'NewMessage'] = improved
                improved_count += 1
                print(f"  Improved: {improved}")
            else:
                failed_count += 1
                print(f"  Failed to improve")
    except KeyboardInterrupt:
        print("\n\nInterrupted by user. Saving partial results...")

    print("\n" + "=" * 80)
    print(f"Summary:")
    print(f"  Total entries: {total}")
    print(f"  Improved: {improved_count}")
    print(f"  Skipped: {skipped_count}")
    print(f"  Failed: {failed_count}")

    # Save the updated CSV
    print(f"\nSaving to {args.output_csv}...")
    try:
        df.to_csv(args.output_csv, index=False)
        print(f"✓ Successfully saved to {args.output_csv}")
    except Exception as e:
        print(f"Error saving CSV: {e}")
        sys.exit(1)

    print("\nNext steps:")
    print(f"1. Review {args.output_csv} to verify the improvements")
    print(f"2. Run: ./logrefactor transform -input {args.output_csv} -path ./your-project -dry-run")
    print(f"3. If satisfied, run: ./logrefactor transform -input {args.output_csv} -path ./your-project")


if __name__ == "__main__":
    main()