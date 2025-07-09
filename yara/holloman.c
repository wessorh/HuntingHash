/*
Copyright (c) 2014. The YARA Authors. All Rights Reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation and/or
other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software without
specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

#include <jansson.h>
#include <yara/endian.h>
#include <yara/globals.h>
#include <yara/mem.h>
#include <yara/modules.h>
#include <yara/strutils.h>

#include <curl/curl.h>
#include <sys/stat.h>

#define HOLLOMAN_LEN 40

#define MODULE_NAME holloman
#define DEFAULT_URL_ENDPOINT "http://localhost:50005/holloman/v2/hh128"

typedef struct _CACHE_KEY
{
  int64_t offset;
  int64_t length;

} CACHE_KEY;

static void digest_to_ascii(
    unsigned char* digest,
    char* digest_ascii,
    size_t digest_length)
{
  size_t i;

  for (i = 0; i < digest_length; i++)
    sprintf(digest_ascii + (i * 2), "%02x", digest[i]);

  digest_ascii[digest_length * 2] = '\0';
}

static char* get_from_cache(
    YR_OBJECT* module_object,
    const char* ns,
    int64_t offset,
    int64_t length)
{
  CACHE_KEY key;
  YR_HASH_TABLE* hash_table = (YR_HASH_TABLE*) module_object->data;

  key.offset = offset;
  key.length = length;

  char* result = (char*) yr_hash_table_lookup_raw_key(
      hash_table, &key, sizeof(key), ns);

  YR_DEBUG_FPRINTF(
      2,
      stderr,
      "- %s(offset=%" PRIi64 " length=%" PRIi64 ") {} = %p\n",
      __FUNCTION__,
      offset,
      length,
      result);

  return result;
}

static int add_to_cache(
    YR_OBJECT* module_object,
    const char* ns,
    int64_t offset,
    int64_t length,
    const char* digest)
{
  CACHE_KEY key;
  YR_HASH_TABLE* hash_table = (YR_HASH_TABLE*) module_object->data;

  char* copy = yr_strdup(digest);

  key.offset = offset;
  key.length = length;

  if (copy == NULL)
    return ERROR_INSUFFICIENT_MEMORY;

  int result = yr_hash_table_add_raw_key(
      hash_table, &key, sizeof(key), ns, (void*) copy);

  YR_DEBUG_FPRINTF(
      2,
      stderr,
      "- %s(offset=%" PRIi64 " length=%" PRIi64 " digest=%s) {} = %d\n",
      __FUNCTION__,
      offset,
      length,
      digest,
      result);

  return result;
}

struct string {
  char *ptr;
  size_t len;
};

void init_string(struct string *s) {
  s->len = 0;
  s->ptr = malloc(s->len+1);
  if (s->ptr == NULL) {
    fprintf(stderr, "malloc() failed\n");
    exit(EXIT_FAILURE);
  }
  s->ptr[0] = '\0';
}

size_t writefunc(void *ptr, size_t size, size_t nmemb, struct string *s)
{
  size_t new_len = s->len + size*nmemb;
  s->ptr = realloc(s->ptr, new_len+1);
  if (s->ptr == NULL) {
    fprintf(stderr, "realloc() failed\n");
    exit(EXIT_FAILURE);
  }
  memcpy(s->ptr+s->len, ptr, size*nmemb);
  s->ptr[new_len] = '\0';
  s->len = new_len;

  return size*nmemb;
}

define_function(string_holloman)
{
//  unsigned char digest[HOLLOMAN_LEN];
  char digest_ascii[HOLLOMAN_LEN];
  SIZED_STRING* s = sized_string_argument(1);
  struct string s2;
  init_string(&s2);

  
  CURL *easy = curl_easy_init();
  curl_mime *mime;
  curl_mimepart *part;
  CURLcode rc;

  /* Build an HTTP form with a single field named "data", */
  mime = curl_mime_init(easy);
  part = curl_mime_addpart(mime);
  curl_mime_data(part, s->c_string, s->length);
  curl_mime_name(part, "holloman-data");
  curl_mime_type(part, "application/octet-stream");
 
  curl_easy_setopt(easy, CURLOPT_WRITEFUNCTION, writefunc);
  curl_easy_setopt(easy, CURLOPT_WRITEDATA, &s2);

  /* Post and send it. */
  curl_easy_setopt(easy, CURLOPT_MIMEPOST, mime);

// enable URL to be set via an environment variable
  curl_easy_setopt(easy, CURLOPT_URL, "http://192.168.8.101:50005/holloman/v2/hh128");
  rc = curl_easy_perform(easy);
 
  fprintf(stderr, "result: %s\n", s2.ptr);
  return_string(s2.ptr)
  /* Clean-up. */
  free(s2.ptr);
  

  curl_easy_cleanup(easy);
  curl_mime_free(mime);  
  
}


define_function(data_holloman)
{

  int past_first_block = false;
  char* buffer; // buffer to hold the object
  
  YR_OBJECT* module = yr_module();

  int64_t arg_offset = integer_argument(1);  // offset where to start
  int64_t arg_length = integer_argument(2);  // length of bytes we want hash on

  int64_t offset = arg_offset;
  int64_t length = arg_length;

  YR_SCAN_CONTEXT* context = yr_scan_context();
  YR_MEMORY_BLOCK* block = first_memory_block(context);
  YR_MEMORY_BLOCK_ITERATOR* iterator = context->iterator;

  // allocate a buffer to contain the data
  buffer = calloc(arg_length, sizeof(char)) ;

  YR_DEBUG_FPRINTF(
      2,
      stderr,
      "+ %s(offset=%" PRIi64 " length=%" PRIi64 ") {\n",
      __FUNCTION__,
      offset,
      length);

  if (block == NULL)
  {
    YR_DEBUG_FPRINTF(
        2, stderr, "} // %s() = YR_UNDEFINED // block == NULL\n", __FUNCTION__);

    return_string(YR_UNDEFINED);
  }

  if (offset < 0 || length < 0 || offset < block->base)
  {
    YR_DEBUG_FPRINTF(
        2,
        stderr,
        "} // %s() = YR_UNDEFINED // bad offset / length\n",
        __FUNCTION__);

    return_string(YR_UNDEFINED);
  }

  foreach_memory_block(iterator, block)
  {
    // if desired block within current block
    if (offset >= block->base && offset < block->base + block->size)
    {
      const uint8_t* block_data = yr_fetch_block_data(block);

      if (block_data != NULL)
      {
        size_t data_offset = (size_t) (offset - block->base);
        size_t data_len = (size_t) yr_min(
            length, (size_t) block->size - data_offset);

        offset += data_len;
        length -= data_len;

        memcpy(buffer, block_data + data_offset, data_len) ;
        // yr_sha1_update(&sha_context, block_data + data_offset, data_len);
      }

      past_first_block = true;
    }
    else if (past_first_block)
    {
      // If offset is not within current block and we already
      // past the first block then the we are trying to compute
      // the checksum over a range of non contiguous blocks. As
      // range contains gaps of undefined data the checksum is
      // undefined.

      YR_DEBUG_FPRINTF(
          2,
          stderr,
          "} // %s() = YR_UNDEFINED // past_first_block\n",
          __FUNCTION__);

      // yr_sha1_final(digest, &sha_context);
      return_string(YR_UNDEFINED);
    }

    if (block->base + block->size >= offset + length)
      break;
  }


  if (!past_first_block)
  {
    YR_DEBUG_FPRINTF(
        2,
        stderr,
        "} // %s() = YR_UNDEFINED // !past_first_block\n",
        __FUNCTION__);

    return_string(YR_UNDEFINED);
  }

  // make a REST request with the object and return the result.
  struct string s2;
  init_string(&s2);

  
  CURL *easy = curl_easy_init();
  curl_mime *mime;
  curl_mimepart *part;
  CURLcode rc;

  /* Build an HTTP form with a single field named "data", */
  mime = curl_mime_init(easy);
  part = curl_mime_addpart(mime);
  curl_mime_data(part, buffer, arg_length);
  curl_mime_name(part, "holloman-data");
  curl_mime_type(part, "application/octet-stream");
  curl_mime_filename(part, "holloman-data");

  curl_easy_setopt(easy, CURLOPT_WRITEFUNCTION, writefunc);
  curl_easy_setopt(easy, CURLOPT_WRITEDATA, &s2);

  /* Post and send it. */
  curl_easy_setopt(easy, CURLOPT_MIMEPOST, mime);

  // enable URL to be set via an environment variable /holloman/v2/hh128
  curl_easy_setopt(easy, CURLOPT_URL, "http://192.168.8.101:50005/holloman/v2/hh128");
  rc = curl_easy_perform(easy);

  //yr_set_string("test", module, "REST_RESULT");
  YR_DEBUG_FPRINTF(2, stderr, "- %s() %s\n", __FUNCTION__, s2.ptr);
  json_error_t json_error;
  json_t* json;

  json = json_loadb(s2.ptr, s2.len,
#if JANSSON_VERSION_HEX >= 0x020600
      JSON_ALLOW_NUL,
#else
      0,
#endif
      &json_error);

  if (json == NULL){
    YR_DEBUG_FPRINTF(2, stderr, "- %s() %s\n", __FUNCTION__, json_error.text);
    return ERROR_INVALID_MODULE_DATA;
  }

  //module_object->data = (void*) json;
  char* id = json_object_get(json, "Id") ;
  char* clid;
  if(id)
    clid = json_string_value(id) ;

  YR_DEBUG_FPRINTF(2, stderr, "clid: %s\n", clid);

  /* Clean-up. */
  free(s2.ptr);
  free(buffer);


  if(clid)
    return_string(clid)
}

begin_declarations
  declare_function("hh128", "ii", "s", data_holloman);
  // declare_function("hh128", "s", "s", string_holloman);
  declare_string("URL_ENDPOINT");
  declare_string("CLIENT_VERSION");
  declare_string("CLID");
  declare_string("REST_RESULT");
end_declarations

int module_initialize(YR_MODULE* module)
{
  YR_DEBUG_FPRINTF(2, stderr, "- %s() {}\n", __FUNCTION__);
  // connect


  return ERROR_SUCCESS;
}

int module_finalize(YR_MODULE* module)
{
  YR_DEBUG_FPRINTF(2, stderr, "- %s() {}\n", __FUNCTION__);

  // disconnect
  

  return ERROR_SUCCESS;
}

int module_load(
    YR_SCAN_CONTEXT* context,
    YR_OBJECT* module_object,
    void* module_data,
    size_t module_data_size)
{
  YR_DEBUG_FPRINTF(2, stderr, "- %s() {}\n", __FUNCTION__);

  YR_HASH_TABLE* hash_table;

  FAIL_ON_ERROR(yr_hash_table_create(17, &hash_table));

  module_object->data = hash_table;
  yr_set_string("v2", module_object, "CLIENT_VERSION");
  char* url = getenv("HOLLOMAN_URL_ENDPOINT");
  if(url)
    yr_set_string(url, module_object ,"URL_ENDPOINT" );

  return ERROR_SUCCESS;
}

int module_unload(YR_OBJECT* module_object)
{
  YR_DEBUG_FPRINTF(2, stderr, "- %s() {}\n", __FUNCTION__);

  YR_HASH_TABLE* hash_table = (YR_HASH_TABLE*) module_object->data;

  if (hash_table != NULL)
    yr_hash_table_destroy(hash_table, (YR_HASH_TABLE_FREE_VALUE_FUNC) yr_free);

  return ERROR_SUCCESS;
}

#undef holloman 
