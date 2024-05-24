import re
import argparse


def extract_go_function(file_path, function_name):
    # Regular expression to match a specific Go function definition
    func_pattern = re.compile(
        rf"func\s+{function_name}\s*\((?P<params>[^)]*)\)\s*(?P<return>.*?){{(?P<body>.*?)^\}}",
        re.DOTALL | re.MULTILINE,
    )

    with open(file_path, "r") as file:
        content = file.read()

    match = func_pattern.search(content)
    if match:
        func_params = match.group("params")
        func_return = match.group("return").strip()
        func_body = match.group("body").strip()

        function_info = {
            "name": function_name,
            "params": func_params,
            "return": func_return,
            "body": func_body,
        }

        return function_info
    else:
        return None


def main():
    parser = argparse.ArgumentParser(
        description="Extract a function from a Go source file."
    )
    parser.add_argument("file_path", type=str, help="Path to the Go source file")
    parser.add_argument(
        "function_name", type=str, help="Name of the function to extract"
    )

    args = parser.parse_args()

    function_info = extract_go_function(args.file_path, args.function_name)
    if function_info:
        print(f"Function Name: {function_info['name']}")
        print(f"Parameters: {function_info['params']}")
        print(f"Return Type: {function_info['return']}")
        print(f"Body:\n{function_info['body']}")
    else:
        print(f"Function '{args.function_name}' not found in the file.")


if __name__ == "__main__":
    main()
