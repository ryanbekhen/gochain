package prompt

var FunctionsToCall = `
You have access to the following tools:

{functions_to_call}

You must always select one of the above tools and respond with only a JSON object matching the following schema:

{
	"tool": <name of the selected tool>,
	"tool_input": <parameters for the selected tool, matching the tool's JSON schema>
}

`
