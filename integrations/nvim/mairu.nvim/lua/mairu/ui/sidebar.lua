local Split = require("nui.split")
local api = require("mairu.api")
local config = require("mairu.config")

local M = {}
local sidebar = nil
local last_query = ""
local timer = nil

local function create_sidebar()
  return Split({
    relative = "editor",
    position = "right",
    size = config.options.ambient.width,
    buf_options = {
      modifiable = true,
      readonly = false,
      filetype = "markdown",
    },
    win_options = {
      winhighlight = "Normal:Normal,FloatBorder:FloatBorder",
      wrap = true,
    },
  })
end

local function format_ambient_results(data, query)
  local lines = { "### Ambient Context", "*Query: " .. query .. "*", "" }
  
  if data.nodes and #data.nodes > 0 then
    table.insert(lines, "**Nodes**")
    for i, item in ipairs(data.nodes) do
      if i > 3 then break end -- Limit to top 3
      local node = item.node or item
      table.insert(lines, "- " .. (node.name or "Unnamed"))
      if node.abstract then
        local abs = node.abstract:sub(1, 100) .. (string.len(node.abstract) > 100 and "..." or "")
        table.insert(lines, "  > " .. abs:gsub("\n", " "))
      end
    end
    table.insert(lines, "")
  end
  
  if data.memories and #data.memories > 0 then
    table.insert(lines, "**Memories**")
    for i, item in ipairs(data.memories) do
      if i > 5 then break end -- Limit to top 5
      local mem = item.memory or item
      table.insert(lines, "- " .. (mem.content or ""))
    end
  end

  if #lines == 3 then
    table.insert(lines, "No context found.")
  end

  return lines
end

function M.update()
  if not sidebar or not sidebar.bufnr or not vim.api.nvim_buf_is_valid(sidebar.bufnr) then
    return
  end
  
  -- Use current file path as query
  local current_file = vim.fn.expand("%:p")
  if current_file == "" then return end
  
  -- Basic project relative path
  local cwd = vim.fn.getcwd()
  local rel_path = current_file:gsub("^" .. vim.pesc(cwd) .. "/", "")
  
  local query = "contextfs://" .. config.options.server.project .. "/" .. rel_path
  
  -- Don't search if it hasn't changed
  if query == last_query then return end
  last_query = query
  
  api.search(query, function(data, err)
    if not sidebar or not sidebar.bufnr or not vim.api.nvim_buf_is_valid(sidebar.bufnr) then return end
    
    if err then return end
    
    local lines = format_ambient_results(data, rel_path)
    
    vim.api.nvim_set_option_value("modifiable", true, { buf = sidebar.bufnr })
    vim.api.nvim_set_option_value("readonly", false, { buf = sidebar.bufnr })
    
    vim.api.nvim_buf_set_lines(sidebar.bufnr, 0, -1, false, lines)
    
    vim.api.nvim_set_option_value("modifiable", false, { buf = sidebar.bufnr })
    vim.api.nvim_set_option_value("readonly", true, { buf = sidebar.bufnr })
  end)
end

function M.toggle()
  if sidebar then
    sidebar:unmount()
    sidebar = nil
    last_query = ""
  else
    sidebar = create_sidebar()
    sidebar:mount()
    
    sidebar:map("n", "q", function()
      M.toggle()
    end, { noremap = true })
    
    M.update()
  end
end

function M.setup_autocmds()
  local group = vim.api.nvim_create_augroup("MairuAmbient", { clear = true })
  
  vim.api.nvim_create_autocmd({"BufEnter"}, {
    group = group,
    callback = function()
      if not sidebar or not config.options.ambient.enabled then return end
      
      -- Debounce
      if timer then timer:stop() timer:close() timer = nil end
      
      local uv = vim.uv or vim.loop
      timer = uv.new_timer()
      timer:start(config.options.ambient.debounce_ms, 0, vim.schedule_wrap(function()
        M.update()
      end))
    end
  })
end

return M
