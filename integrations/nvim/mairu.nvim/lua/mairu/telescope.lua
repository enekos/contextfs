local pickers = require("telescope.pickers")
local finders = require("telescope.finders")
local conf = require("telescope.config").values
local actions = require("telescope.actions")
local action_state = require("telescope.actions.state")
local api = require("mairu.api")

local M = {}

local function store_memory()
  vim.ui.input({ prompt = "Memory to store: " }, function(input)
    if not input or input == "" then return end
    
    api.store_memory(input, function(data, err)
      if err then
        vim.notify("Failed to store memory: " .. err, vim.log.levels.ERROR, { title = "Mairu" })
      else
        vim.notify("Memory stored successfully.", vim.log.levels.INFO, { title = "Mairu" })
      end
    end)
  end)
end

local function search_context()
  vim.ui.input({ prompt = "Search Query: " }, function(input)
    if not input or input == "" then return end
    
    require("mairu.ui.popup").open_search(input)
  end)
end

local function vibe_query()
  vim.ui.input({ prompt = "Vibe Query: " }, function(input)
    if not input or input == "" then return end
    
    -- Open chat window and seed the query
    require("mairu.ui.chat").open()
    -- Note: ideally we'd pre-fill the chat input here or send it directly.
    -- For simplicity we just open chat. The user can type it.
  end)
end

local function vibe_mutation()
  vim.ui.input({ prompt = "Mutation Plan: " }, function(input)
    if not input or input == "" then return end
    
    api.vibe_mutation(input, function(data, err)
      if err then
        vim.notify("Failed to execute mutation: " .. err, vim.log.levels.ERROR, { title = "Mairu" })
      else
        vim.notify("Mutation applied successfully: " .. (data.message or "Done"), vim.log.levels.INFO, { title = "Mairu" })
      end
    end)
  end)
end

local function find_symbol_impact()
  vim.ui.input({ prompt = "Symbol or Node to find blast radius for: " }, function(input)
    if not input or input == "" then return end
    require("mairu.ui.popup").open_search(input) -- For now, standard search
  end)
end

function M.command_palette(opts)
  opts = opts or {}
  
  local commands = {
    { display = "Search Context (Contextual Search)", cmd = search_context },
    { display = "Find Symbol Impact (Blast Radius)", cmd = find_symbol_impact },
    { display = "Store Memory (Natural Language)", cmd = store_memory },
    { display = "Ask Mairu (Vibe Query)", cmd = vibe_query },
    { display = "Mairu Update Knowledge (Vibe Mutation)", cmd = vibe_mutation },
    { display = "Toggle Ambient Context Sidebar", cmd = require("mairu.ui.sidebar").toggle },
    { display = "Open Mairu Chat", cmd = require("mairu.ui.chat").open },
  }
  
  pickers.new(opts, {
    prompt_title = "Mairu Commands",
    finder = finders.new_table({
      results = commands,
      entry_maker = function(entry)
        return {
          value = entry,
          display = entry.display,
          ordinal = entry.display,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection then
          selection.value.cmd()
        end
      end)
      return true
    end,
  }):find()
end

return M
