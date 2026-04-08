local Popup = require("nui.popup")
local event = require("nui.utils.autocmd").event
local api = require("mairu.api")

local M = {}

local function create_popup()
  return Popup({
    enter = true,
    focusable = true,
    border = {
      style = "rounded",
      text = {
        top = " Mairu Context Search ",
        top_align = "center",
      },
    },
    position = "50%",
    size = {
      width = "80%",
      height = "60%",
    },
    buf_options = {
      modifiable = true,
      readonly = false,
      filetype = "markdown",
    },
    win_options = {
      winblend = 10,
      winhighlight = "Normal:Normal,FloatBorder:FloatBorder",
    },
  })
end

local function format_results(data)
  local lines = {}
  
  -- Handle Context Nodes
  if data.contextNodes and #data.contextNodes > 0 then
    table.insert(lines, "# Context Nodes")
    table.insert(lines, "")
    for _, item in ipairs(data.contextNodes) do
      local node = item.node or item -- Handle potential wrapper
      table.insert(lines, "## " .. (node.name or "Unnamed Node"))
      table.insert(lines, "**URI:** `" .. (node.uri or "") .. "`")
      if node.abstract and node.abstract ~= "" then
        table.insert(lines, "")
        for _, line in ipairs(vim.split(node.abstract, "\n")) do
          table.insert(lines, "> " .. line)
        end
      end
      if node.overview and node.overview ~= "" then
        table.insert(lines, "")
        table.insert(lines, "```typescript") -- Or try to guess from URI
        for _, line in ipairs(vim.split(node.overview, "\n")) do
          table.insert(lines, line)
        end
        table.insert(lines, "```")
      end
      table.insert(lines, "")
    end
    table.insert(lines, "---")
    table.insert(lines, "")
  end
  
  -- Handle Memories
  if data.memories and #data.memories > 0 then
    table.insert(lines, "# Memories")
    table.insert(lines, "")
    for _, item in ipairs(data.memories) do
      local mem = item.memory or item
      table.insert(lines, "- " .. (mem.content or ""))
      if mem.category then
         table.insert(lines, "  *(" .. mem.category .. ")*")
      end
    end
  end

  if #lines == 0 then
    table.insert(lines, "No context found.")
  end

  return lines
end

function M.open_search_for_cursor()
  local word = vim.fn.expand('<cword>')
  if not word or word == "" then
    vim.notify("No word under cursor.", vim.log.levels.WARN, { title = "Mairu" })
    return
  end
  
  M.open_search(word)
end

function M.open_search(query)
  local popup = create_popup()
  
  -- Unmount on Esc or q
  popup:map("n", "q", function()
    popup:unmount()
  end, { noremap = true })
  
  popup:map("n", "<Esc>", function()
    popup:unmount()
  end, { noremap = true })
  
  popup:mount()
  
  -- Loading state
  vim.api.nvim_buf_set_lines(popup.bufnr, 0, -1, false, { "Searching for: " .. query .. "..." })

  api.search(query, function(data, err)
    if not popup.bufnr or not vim.api.nvim_buf_is_valid(popup.bufnr) then
      return
    end
    
    if err then
      vim.api.nvim_buf_set_lines(popup.bufnr, 0, -1, false, { "Error: " .. err })
      return
    end
    
    local lines = format_results(data)
    vim.api.nvim_buf_set_lines(popup.bufnr, 0, -1, false, lines)
    
    -- Make readonly after setting lines
    vim.api.nvim_set_option_value("modifiable", false, { buf = popup.bufnr })
    vim.api.nvim_set_option_value("readonly", true, { buf = popup.bufnr })
  end)
end

return M
