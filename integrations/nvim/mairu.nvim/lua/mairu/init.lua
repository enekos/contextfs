local config = require("mairu.config")
local server = require("mairu.server")

local M = {}

function M.setup(opts)
  -- 1. Setup config
  config.setup(opts)
  
  -- 2. Start server if configured
  if config.options.server.auto_start then
    server.start()
    
    -- Ensure server stops when neovim exits
    vim.api.nvim_create_autocmd("VimLeavePre", {
      callback = function()
        server.stop()
      end,
    })
  end
  
  -- 3. Setup Ambient Context Sidebar autocmds
  if config.options.ambient.enabled then
    require("mairu.ui.sidebar").setup_autocmds()
  end
  
  -- 4. Setup user commands
  vim.api.nvim_create_user_command("MairuSearch", function(args)
    if args.args and args.args ~= "" then
      require("mairu.ui.popup").open_search(args.args)
    else
      require("mairu.ui.popup").open_search_for_cursor()
    end
  end, { nargs = "?" })
  
  vim.api.nvim_create_user_command("MairuCommands", function()
    require("mairu.telescope").command_palette()
  end, {})
  
  vim.api.nvim_create_user_command("MairuSidebar", function()
    require("mairu.ui.sidebar").toggle()
  end, {})
  
  vim.api.nvim_create_user_command("MairuChat", function()
    require("mairu.ui.chat").open()
  end, {})
  
  -- 5. Set default keymaps (can be overridden by user)
  -- We don't forcefully map unless user asks, but we provide a default mapper function
end

function M.set_default_keymaps()
  vim.keymap.set("n", "<leader>ms", function()
    require("mairu.ui.popup").open_search_for_cursor()
  end, { desc = "Mairu Search (Cursor)" })
  
  vim.keymap.set("v", "<leader>ms", function()
    vim.cmd('noau normal! "vy"')
    local text = vim.fn.getreg('v')
    require("mairu.ui.popup").open_search(text)
  end, { desc = "Mairu Search (Selection)" })
  
  vim.keymap.set("n", "<leader>mc", ":MairuCommands<CR>", { desc = "Mairu Command Palette" })
  vim.keymap.set("n", "<leader>ma", ":MairuChat<CR>", { desc = "Mairu Chat" })
  vim.keymap.set("n", "<leader>mb", ":MairuSidebar<CR>", { desc = "Mairu Sidebar Toggle" })
end

return M
