local curl = require("plenary.curl")
local config = require("mairu.config")

local M = {}

local function get_base_url()
  return "http://localhost:" .. config.options.server.port .. "/api"
end

local function make_request(opts)
  local url = get_base_url() .. opts.path
  
  local curl_opts = {
    url = url,
    method = opts.method or "GET",
    headers = {
      ["Content-Type"] = "application/json",
    },
    timeout = config.options.api.timeout,
    callback = vim.schedule_wrap(function(res)
      if not res then
        opts.callback(nil, "Network error: curl request failed")
        return
      end
      
      local ok, decoded = pcall(vim.fn.json_decode, res.body)
      if res.status >= 200 and res.status < 300 then
        if ok then
          opts.callback(decoded, nil)
        else
          opts.callback(nil, "Failed to decode JSON response")
        end
      else
        local err_msg = "API Error (" .. res.status .. ")"
        if ok and decoded and decoded.error then
          err_msg = err_msg .. ": " .. decoded.error
        end
        opts.callback(nil, err_msg)
      end
    end)
  }

  if opts.body then
    curl_opts.body = vim.fn.json_encode(opts.body)
  end

  if opts.query then
    curl_opts.query = opts.query
  end

  curl.request(curl_opts)
end

function M.search(query, callback)
  make_request({
    path = "/search",
    method = "GET",
    query = {
      q = query,
      project = config.options.server.project,
      topK = "5"
    },
    callback = callback
  })
end

function M.store_memory(content, callback)
  make_request({
    path = "/memories",
    method = "POST",
    body = {
      content = content,
      project = config.options.server.project,
      owner = "agent",
      category = "observation",
      importance = 5
    },
    callback = callback
  })
end

function M.vibe_mutation(prompt, callback)
  -- Step 1: Plan
  make_request({
    path = "/vibe/mutation/plan",
    method = "POST",
    body = {
      prompt = prompt,
      project = config.options.server.project,
    },
    callback = function(plan_data, plan_err)
      if plan_err then
        callback(nil, "Plan failed: " .. plan_err)
        return
      end
      
      -- Step 2: Auto-execute
      make_request({
        path = "/vibe/mutation/execute",
        method = "POST",
        body = {
          operations = plan_data.operations,
          project = config.options.server.project,
        },
        callback = callback
      })
    end
  })
end

function M.autocomplete(opts, callback)
  make_request({
    path = "/autocomplete",
    method = "POST",
    body = {
      prefix = opts.prefix,
      suffix = opts.suffix,
      filename = opts.filename,
      project = config.options.server.project,
    },
    callback = callback
  })
end

return M
